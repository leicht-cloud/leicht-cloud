package limiter

import (
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/auth"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type uploadMiddleWare struct {
	handler auth.AuthHandlerInterface

	db *gorm.DB
}

func UploadMiddleware(db *gorm.DB, handler auth.AuthHandlerInterface) auth.AuthHandlerInterface {
	return &uploadMiddleWare{
		handler: handler,
		db:      db,
	}
}

func (m *uploadMiddleWare) Serve(user *models.User, w http.ResponseWriter, r *http.Request) {
	limit := &models.UploadLimit{}
	tx := m.db.First(limit, "user_id = ?", user.ID)

	// if we either can't find an entry for this user, or the unlimited flag is set
	// then we just pass everything along as is
	// TODO: Allow us to set a default
	if tx.Error != nil || limit.Unlimited {
		m.handler.Serve(user, w, r)
		return
	}

	logrus.Debugf("Applying the following rate limits: %#v", limit)

	// otherwise we override the body with the freshly fetched settings
	r.Body = NewReader(r.Body, float64(limit.RateLimit), limit.Burst)

	// and we pass the request with the adjusted body along to the wrapped handler
	m.handler.Serve(user, w, r)
}
