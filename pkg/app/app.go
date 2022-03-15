package app

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
)

type App struct {
	plugin plugin.PluginInterface
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
