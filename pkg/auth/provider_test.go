package auth

import (
	"path/filepath"
	"testing"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProvider(t *testing.T) (*gorm.DB, *Provider) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")))
	if err != nil {
		t.Fatal(err)
	}

	err = models.InitModels(db)
	if err != nil {
		t.Fatal(err)
	}

	return db, NewProvider(db)
}

func TestAuthenticate(t *testing.T) {
	_, provider := setupProvider(t)

	user := &models.User{
		ID:    1,
		Email: "test@test.com",
	}

	key, err := provider.Authenticate(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, key)
}

func TestInvalidToken(t *testing.T) {
	_, provider := setupProvider(t)

	user, err := provider.verifyCookie("This is clearly not valid, lol")
	assert.Nil(t, user)
	assert.Error(t, err)
}

func TestVerify(t *testing.T) {
	db, provider := setupProvider(t)

	user := &models.User{
		ID:    1,
		Email: "test@test.com",
	}

	key, err := provider.Authenticate(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	// this test should fail, as the doesn't exist in the database
	verifiedUser, err := provider.verifyCookie(key)
	assert.Error(t, err)
	assert.Nil(t, verifiedUser)

	assert.NoError(t, db.Begin().Create(&user).Commit().Error)

	verifiedUser, err = provider.verifyCookie(key)
	assert.NoError(t, err)
	if assert.NotNil(t, verifiedUser) {
		assert.Equal(t, user.ID, verifiedUser.ID)
		assert.Equal(t, user.Email, verifiedUser.Email)
	}
}
