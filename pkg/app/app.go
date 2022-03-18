package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
)

type App struct {
	plugin plugin.PluginInterface

	jwtPrivateKey ed25519.PrivateKey
	jwtPublicKey  ed25519.PublicKey
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
	}, nil
}

type UserClaims struct {
	jwt.StandardClaims
	models.User
}

func (a *App) Serve(user *models.User, w http.ResponseWriter, method, path string, body io.Reader) error {
	uri, err := url.Parse(fmt.Sprintf("http://127.0.0.1/%s", path))
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		return err
	}

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

	_, err = io.Copy(w, resp.Body)

	return err
}

func (a *App) IFramePermissions() string {
	manifest := a.plugin.Manifest()
	out := ""

	if manifest.Permissions.App.Javascript {
		out += "allow-scripts "
	}

	return strings.Trim(out, " ")
}
