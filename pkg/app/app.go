package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"go.uber.org/multierr"
)

type App struct {
	plugin plugin.PluginInterface

	jwtPrivateKey ed25519.PrivateKey
	jwtPublicKey  ed25519.PublicKey

	closers []io.Closer
}

func newApp(plugin plugin.PluginInterface) (*App, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &App{
		plugin:        plugin,
		jwtPrivateKey: privateKey,
		jwtPublicKey:  publicKey,
		closers:       make([]io.Closer, 0),
	}, nil
}

func (a *App) Close() error {
	var err error
	for _, closer := range a.closers {
		err = multierr.Append(err, closer.Close())
	}

	return err
}

func (a *App) GetPlugin() plugin.PluginInterface {
	return a.plugin
}

type UserClaims struct {
	jwt.StandardClaims
	models.User
}

func (a *App) Serve(user *models.User, w http.ResponseWriter, method, path string, query url.Values, headers http.Header, body io.Reader) error {
	rawUri := fmt.Sprintf("http://127.0.0.1/%s", strings.TrimPrefix(path, "/"))
	if len(query) > 0 {
		rawUri = fmt.Sprintf("%s?%s", rawUri, query.Encode())
	}

	uri, err := url.Parse(rawUri)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		return err
	}

	req.Header = headers

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, UserClaims{
		User: *user,
	})

	// Sign and get the complete encoded token as a string using the secret
	userToken, err := token.SignedString(a.jwtPrivateKey)
	if err != nil {
		return err
	}

	req.Header.Add("X-Leicht-Cloud-User", userToken)

	resp, err := a.plugin.HttpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// if our status code range is in the 300, we have to manually extract and rewrite the redirect
	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		// check if there's an actual location header
		location := resp.Header.Get("Location")
		if location == "" {
			return fmt.Errorf("No Location header")
		}

		// and redirect to it
		// TODO: Automatically add /apps/embed/<app>/ in front of it
		http.Redirect(w, req, location, resp.StatusCode)
	} else {
		// plain proxy the status code
		w.WriteHeader(resp.StatusCode)

		// and copy the entire body
		_, err = io.Copy(w, resp.Body)
	}

	return err
}

func (a *App) IFramePermissions() string {
	manifest := a.plugin.Manifest()
	out := ""

	if manifest.Permissions.App.Javascript {
		out += "allow-scripts "
	}

	if manifest.Permissions.App.Forms {
		out += "allow-forms "
	}

	return strings.Trim(out, " ")
}

var ErrNoMatch = errors.New("No match")

func (a *App) Opener(mime types.MimeType) (string, error) {
	openers := a.plugin.Manifest().Permissions.App.FileOpener

	for pattern, path := range openers {
		if mime.Match(pattern) {
			return path, nil
		}
	}

	return "", ErrNoMatch
}
