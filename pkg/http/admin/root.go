package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/schoentoon/go-cloud/pkg/auth"
	"github.com/schoentoon/go-cloud/pkg/http/template"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type rootHandler struct {
	StaticHandler http.Handler
	PluginManager *plugin.Manager
	DB            *gorm.DB
}

type adminTemplateData struct {
	Navbar  template.NavbarData
	Plugins []string

	Page       string
	PluginView pluginView
	UserList   userList
	UserView   userView
}

type pluginView struct {
	Name string
}

type userList struct {
	Users []*models.User
}

type userView struct {
	User             models.User
	UploadLimit      models.UploadLimit
	UploadLimitHuman struct {
		Number int64
		Metric string
	}
}

func newUserList(db *gorm.DB) (userList, error) {
	out := userList{}

	tx := db.Find(&out.Users)

	return out, tx.Error
}

func newUserView(db *gorm.DB, rawID string) (userView, error) {
	out := userView{}
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return out, err
	}

	tx := db.Find(&out.User, id)
	if tx.Error != nil {
		return out, tx.Error
	}

	db.First(&out.UploadLimit, "user_id = ?", out.User.ID)

	if out.UploadLimit.RateLimit > 0 && !out.UploadLimit.Unlimited {
		kilobytes := out.UploadLimit.RateLimit / 1024
		if kilobytes > 1024 {
			megabytes := kilobytes / 1024
			out.UploadLimitHuman.Number = megabytes
			out.UploadLimitHuman.Metric = "mbps"
		} else {
			out.UploadLimitHuman.Number = kilobytes
			out.UploadLimitHuman.Metric = "kbps"
		}
	}

	return out, nil
}

func (h *rootHandler) handlePost(r *http.Request) error {
	// FIXME: this currently completely assumes you're in /admin/user/
	// this should be fixed by having separate handler for the different admin pages, lol
	rawID := strings.TrimPrefix(r.URL.Path, "/admin/user/")
	user := &models.User{}
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return err
	}

	tx := h.DB.Find(&user, id)
	if tx.Error != nil {
		return tx.Error
	}

	err = r.ParseForm()
	if err != nil {
		return err
	}

	if r.Form.Has("upload_limit_number") && r.Form.Has("upload_limit_metric") {
		number, err := strconv.ParseInt(r.FormValue("upload_limit_number"), 10, 64)
		if err != nil {
			return err
		}
		switch r.FormValue("upload_limit_metric") {
		case "kbps":
			number *= 1024
		case "mbps":
			number *= 1024 * 1024
		case "unlimited":
			number = 0
		}

		upload_limit := models.UploadLimit{
			UserID:    user.ID,
			User:      user,
			Unlimited: number == 0,
			RateLimit: number,
			Burst:     number,
		}

		tx = h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			UpdateAll: true,
		}).Create(&upload_limit)
		if tx.Error != nil {
			return tx.Error
		}
	}

	return nil
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil || !user.Admin {
		// we redirect you to / as you don't have permission to view the admin panel
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if r.Method == http.MethodPost {
		err := h.handlePost(r)
		if err != nil {
			logrus.Error(err)
		}
	}

	var err error
	data := adminTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
		Plugins: h.PluginManager.Plugins(),
	}

	// TODO: Clean this up into seperate handlers and share the side bar elsewhere...
	if strings.HasPrefix(r.URL.Path, "/admin/plugin/") {
		data.Page = "plugin"
		data.PluginView = pluginView{
			Name: strings.TrimPrefix(r.URL.Path, "/admin/plugin/"),
		}
	} else if strings.HasPrefix(r.URL.Path, "/admin/userlist") {
		data.Page = "userlist"
		data.UserList, err = newUserList(h.DB)
	} else if strings.HasPrefix(r.URL.Path, "/admin/user/") {
		data.Page = "user"
		data.UserView, err = newUserView(h.DB, strings.TrimPrefix(r.URL.Path, "/admin/user/"))
		logrus.Debugf("%#v", data.UserView)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// internal rewrite to admin page, so we render that
	r.URL.Path = "/admin.gohtml"

	ctx := template.AttachTemplateData(r.Context(), data)

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}
