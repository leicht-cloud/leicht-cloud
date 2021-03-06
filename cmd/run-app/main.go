package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/leicht-cloud/leicht-cloud/pkg/app"
	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	lchttp "github.com/leicht-cloud/leicht-cloud/pkg/http"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/builtin/local"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var dummyUser = &models.User{
	ID:    1,
	Email: "test@test.com",
}

func main() {
	logrus.SetReportCaller(true)
	runtime := flag.String("runtime", "namespace", "The kind of runtime to use for the app")

	flag.Parse()

	if len(flag.Args()) != 1 {
		logrus.Fatalf("Requires 1 argument, got %d", len(flag.Args()))
	}
	path := flag.Arg(0)
	appname := strings.TrimSuffix(filepath.Base(path), ".plugin")
	tmpdir, err := os.MkdirTemp(os.TempDir(), "leicht-cloud-*")
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("Setting up tmp directory: %s", tmpdir)
	defer os.RemoveAll(tmpdir)

	err = os.Mkdir(filepath.Join(tmpdir, "data"), 0700)
	if err != nil {
		logrus.Fatal(err)
	}

	_, authProvider, err := setupAuthProvider(filepath.Join(tmpdir, "test.db"))
	if err != nil {
		logrus.Fatal(err)
	}

	store := local.NewStorageProvider(path)

	config := plugin.Config{
		Debug:   true,
		Path:    []string{filepath.Dir(path)},
		WorkDir: tmpdir,
		Runner:  *runtime,
	}

	pluginManager, err := config.CreateManager(nil)
	if err != nil {
		logrus.Fatal(err)
	}

	appManager, err := app.NewManager(pluginManager, store, nil)
	if err != nil {
		logrus.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/apps/embed/", auth.AuthHandler(appManager))
	assets, err := lchttp.InitStatic()
	if err != nil {
		logrus.Fatal(err)
	}
	mux.Handle("/", &rootHandler{
		Auth:          authProvider,
		AppManager:    appManager,
		StaticHandler: http.FileServer(http.FS(assets)),
		appname:       appname,
	})

	httpServer := http.Server{
		Handler: auth.AuthMiddleware(authProvider, mux),
		Addr:    "127.0.0.1:8080",
	}

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil {
			logrus.Fatal(err)
		}
	}()

	appInstance, err := appManager.StartApp(appname)
	if err != nil {
		logrus.Fatal(err)
	}
	plugin := appInstance.GetPlugin()

	logrus.Info("Switching to plugin stdout")
	stdout := plugin.Stdout()
	go io.Copy(os.Stdout, stdout) // nolint:errcheck

	chFileChange := detectFileChanges(path)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for {
		select {
		case <-chFileChange:
			logrus.Infof("Detected changes in %s, restarting app", path)
			err = appManager.StopApp(appname)
			if err != nil {
				logrus.Fatalf("Failed to stop app: %s", err)
			}
			appInstance, err := appManager.StartApp(appname)
			if err != nil {
				logrus.Fatal(err)
			}
			plugin = appInstance.GetPlugin()

			logrus.Info("Switching to plugin stdout")
			if stdout != nil {
				stdout.Close() // we close the previous stdout, mainly so we don't leak goroutine and memories of previous stdouts
			}
			stdout = plugin.Stdout()
			go io.Copy(os.Stdout, stdout) // nolint:errcheck

		case <-c:
			logrus.Info("Closing server")

			httpServer.Close()
			return
		}
	}
}

func setupAuthProvider(path string) (*gorm.DB, *auth.Provider, error) {
	db, err := gorm.Open(sqlite.Open(path))
	if err != nil {
		return nil, nil, err
	}

	err = models.InitModels(db)
	if err != nil {
		return nil, nil, err
	}

	// we create a temporary key ourselves, rather than let the regular application library do this
	// as the regular application library will print a warning telling the user to add it to their config
	// which makes very little sense for this application.
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	config := &auth.Config{
		PrivateKey: string(privateKey),
	}

	provider, err := config.Create(db)
	if err != nil {
		return nil, nil, err
	}

	err = db.Begin().Create(&dummyUser).Commit().Error
	if err != nil {
		return nil, nil, err
	}

	return db, provider, nil
}

type rootHandler struct {
	Auth          *auth.Provider
	AppManager    *app.Manager
	StaticHandler http.Handler

	appname string
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if auth.GetUserFromRequest(r) == nil {
		token, err := h.Auth.Authenticate(dummyUser)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  token,
			MaxAge: 86400,
		})
	}

	if r.URL.Path != "/" {
		h.StaticHandler.ServeHTTP(w, r)
		return
	}

	app, err := h.AppManager.GetApp(h.appname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, `<html>

	<head>
		<link href="/css/bootstrap.min.css" rel="stylesheet" crossorigin="anonymous">
		<script src="/js/lib/bootstrap.bundle.min.js"></script>
		<script src="/js/lib/jquery.min.js"></script>
	</head>
	
	<body>
		<div style="height:90%%" class="container wrapper">
			<iframe src="/apps/embed/%s/" height="100%%" width="100%%" sandbox="%s">
			</iframe>
		</div>
	</body>
	
	</html>`, h.appname, app.IFramePermissions())
}

func detectFileChanges(path string) <-chan time.Time {
	ch := make(chan time.Time)

	go func(ch chan<- time.Time) {
		stat, err := os.Stat(path)
		if err != nil {
			logrus.Fatalf("file %s doesn't exist?: %s", path, err)
		}
		last := stat.ModTime()
		for range time.Tick(time.Second) {
			stat, err := os.Stat(path)
			if err != nil {
				logrus.Fatalf("plugin gone? Shutting down: %s", err)
			}
			mod := stat.ModTime()
			if mod.After(last) {
				last = mod
				ch <- mod
			}
		}
	}(ch)

	return ch
}
