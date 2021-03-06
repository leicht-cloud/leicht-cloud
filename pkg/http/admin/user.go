package admin

import (
	"net/http"
	"strconv"

	"github.com/leicht-cloud/leicht-cloud/pkg/http/template"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userHandler struct {
	StaticHandler http.Handler
	DB            *gorm.DB
}

type userTemplateData struct {
	Navbar template.NavbarData
	User   models.User

	UploadLimit      models.UploadLimit
	UploadLimitHuman struct {
		Number int64
		Metric string
	}

	DownloadLimit      models.DownloadLimit
	DownloadLimitHuman struct {
		Number int64
		Metric string
	}
}

func (d *userTemplateData) FillUploadLimit(db *gorm.DB) error {
	db.First(&d.UploadLimit, "user_id = ?", d.User.ID)

	if d.UploadLimit.RateLimit > 0 && !d.UploadLimit.Unlimited {
		kilobytes := d.UploadLimit.RateLimit / 1024
		if kilobytes > 1024 {
			megabytes := kilobytes / 1024
			d.UploadLimitHuman.Number = megabytes
			d.UploadLimitHuman.Metric = "mbps"
		} else {
			d.UploadLimitHuman.Number = kilobytes
			d.UploadLimitHuman.Metric = "kbps"
		}
	}

	return nil
}

func (d *userTemplateData) FillDownloadLimit(db *gorm.DB) error {
	db.First(&d.DownloadLimit, "user_id = ?", d.User.ID)

	if d.DownloadLimit.RateLimit > 0 && !d.DownloadLimit.Unlimited {
		kilobytes := d.DownloadLimit.RateLimit / 1024
		if kilobytes > 1024 {
			megabytes := kilobytes / 1024
			d.DownloadLimitHuman.Number = megabytes
			d.DownloadLimitHuman.Metric = "mbps"
		} else {
			d.DownloadLimitHuman.Number = kilobytes
			d.DownloadLimitHuman.Metric = "kbps"
		}
	}

	return nil
}

func (h *userHandler) handlePost(r *http.Request) error {
	user, err := h.GetIntendedUser(r)
	if err != nil {
		return err
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

		tx := h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			UpdateAll: true,
		}).Create(&upload_limit)
		if tx.Error != nil {
			return tx.Error
		}
	}

	if r.Form.Has("download_limit_number") && r.Form.Has("download_limit_metric") {
		number, err := strconv.ParseInt(r.FormValue("download_limit_number"), 10, 64)
		if err != nil {
			return err
		}
		switch r.FormValue("download_limit_metric") {
		case "kbps":
			number *= 1024
		case "mbps":
			number *= 1024 * 1024
		case "unlimited":
			number = 0
		}

		download_limit := models.DownloadLimit{
			UserID:    user.ID,
			User:      user,
			Unlimited: number == 0,
			RateLimit: number,
			Burst:     number,
		}

		tx := h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			UpdateAll: true,
		}).Create(&download_limit)
		if tx.Error != nil {
			return tx.Error
		}
	}

	return nil
}

func (h *userHandler) GetIntendedUser(r *http.Request) (*models.User, error) {
	rawID := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return nil, err
	}

	user := &models.User{}
	tx := h.DB.Find(&user, id)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return user, nil
}

func (h *userHandler) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := h.handlePost(r)
		if err != nil {
			logrus.Error(err)
		}
	}

	data := userTemplateData{
		Navbar: template.NavbarData{
			Admin: user.Admin,
		},
	}

	user, err := h.GetIntendedUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data.User = *user

	err = multierr.Combine(
		data.FillUploadLimit(h.DB),
		data.FillDownloadLimit(h.DB),
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// internal rewrite to admin page, so we render that
	r.URL.Path = "/admin.user.gohtml"

	ctx := template.AttachTemplateData(r.Context(), data)

	h.StaticHandler.ServeHTTP(w, r.WithContext(ctx))
}
