package app

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/prometheus"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/firewall"
	storagePlugin "github.com/leicht-cloud/leicht-cloud/pkg/storage/plugin"
	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
)

type Manager struct {
	apps map[string]*App
}

func NewManager(pManager *plugin.Manager, store storage.StorageProvider, prom *prometheus.Manager, apps ...string) (*Manager, error) {
	out := &Manager{
		apps: make(map[string]*App),
	}

	for _, name := range apps {
		plugin, err := pManager.Start(name, "app")
		if err != nil {
			return nil, err
		}

		app, err := newApp(plugin)
		if err != nil {
			return nil, err
		}

		manifest := plugin.Manifest()

		if manifest.Permissions.App.Storage.Enabled {
			err = app.setupStorage(store, manifest.Permissions.App.Storage.ReadWrite, manifest.Permissions.App.Storage.WholeStore)
			if err != nil {
				return nil, err
			}
		}

		out.apps[name] = app
	}

	return out, nil
}

func (m *Manager) Close() error {
	var err error
	for _, app := range m.apps {
		err = multierr.Append(err, app.Close())
	}
	return err
}

func (a *App) setupStorage(store storage.StorageProvider, readwrite, wholestore bool) error {
	socketPath := filepath.Join(a.plugin.WorkDir(), "storage.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	// we add the listener as a closer, so it will close correctly
	a.closers = append(a.closers, listener)

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024*32),
		grpc.WriteBufferSize(0),
		grpc.ReadBufferSize(0),
	)

	if !readwrite { // if readwrite isn't toggled, we wrap into ReadOnly
		store = storage.ReadOnly(store)
	}
	if !wholestore { // if we're not wholestore we will be restricted to a subfolder in /apps/<appname>
		store = firewall.Firewall(store, fmt.Sprintf("/apps/%s", a.plugin.Manifest().Name))
	}

	storagePlugin.RegisterStorageProviderServer(grpcServer, storagePlugin.NewStorageBridge(store))

	go func(listener net.Listener, grpcServer *grpc.Server) {
		logrus.Infof("Starting storage listener on host on: %s", socketPath)
		err := grpcServer.Serve(listener)
		if err != nil {
			logrus.Error(err)
		}
	}(listener, grpcServer)

	return nil
}

func (m *Manager) GetApp(name string) (*App, error) {
	app, ok := m.apps[name]
	if !ok {
		return nil, fmt.Errorf("App not found: %s", name)
	}

	return app, nil
}

func (m *Manager) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	appname := ""
	path := "/"

	split := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/apps/embed/"), "/", 2)
	if len(split) < 2 {
		http.Error(w, "Not enough parameters", http.StatusBadRequest)
		return
	}

	appname = strings.Trim(split[0], "/")
	if len(split) == 2 {
		path = split[1]
	}

	switch action {
	// in case our action is open, we are being accessed through the "open app" menu in the fileinfo section
	case "open":
		app, err := m.GetApp(appname)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// we're just going to trust the mime type that was passed here
		mime, err := types.ParseMime(r.URL.Query().Get("mime"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// we check for any openers that would match our mime type
		path, err = app.Opener(mime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// and finally we replace %file% in the path string of the opener, with the actual filename
		filename := r.URL.Query().Get("file")
		path = strings.ReplaceAll(path, "%file%", url.QueryEscape(filename))
	}

	app, err := m.GetApp(appname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	err = app.Serve(user, w, r.Method, path, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) Apps() []string {
	out := make([]string, 0)

	for app := range m.apps {
		out = append(out, app)
	}

	return out
}

func (m *Manager) Openers(mime types.MimeType) map[string]string {
	out := make(map[string]string)
	for name, app := range m.apps {
		path, err := app.Opener(mime)
		if err == nil {
			out[name] = path
		}
	}

	return out
}
