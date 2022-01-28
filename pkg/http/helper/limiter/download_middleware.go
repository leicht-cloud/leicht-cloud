package limiter

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type downloadMiddleware struct {
	handler auth.AuthHandlerInterface

	db *gorm.DB
}

func DownloadMiddleware(db *gorm.DB, handler auth.AuthHandlerInterface) auth.AuthHandlerInterface {
	return &downloadMiddleware{
		handler: handler,
		db:      db,
	}
}

func (m *downloadMiddleware) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	limit := &models.DownloadLimit{}
	tx := m.db.First(limit, "user_id = ?", user.ID)

	// if we either can't find an entry for this user, or the unlimited flag is set
	// then we just pass everything along as is
	// TODO: Allow us to set a default
	if tx.Error != nil || limit.Unlimited {
		m.handler.Serve(user, w, r)
		return
	}

	logrus.Debugf("Applying the following rate limits: %#v", limit)

	// and we pass the request with the adjusted body along to the wrapped handler
	m.handler.Serve(user, NewResponseWriter(w, float64(limit.RateLimit), limit.Burst), r)
}
