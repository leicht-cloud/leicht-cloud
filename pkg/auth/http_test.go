package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	db, provider := setupProvider(t)

	user := &models.User{
		ID:    1,
		Email: "test@test.com",
	}

	assert.NoError(t, db.Begin().Create(&user).Commit().Error)

	key, err := provider.Authenticate(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	req := httptest.NewRequest("GET", "http://127.0.0.1/not/relevant", nil)
	req.AddCookie(&http.Cookie{
		Name:   "auth",
		Value:  key,
		MaxAge: 86400,
	})
	w := httptest.NewRecorder()

	assert.Nil(t, GetUserFromRequest(req))

	called := false
	middleware := AuthMiddleware(provider, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true

		requestUser := GetUserFromRequest(r)
		if assert.NotNil(t, requestUser) {
			assert.Equal(t, user.ID, requestUser.ID)
			assert.Equal(t, user.Email, requestUser.Email)
		}
	}))

	middleware.ServeHTTP(w, req)

	assert.True(t, called)
}
