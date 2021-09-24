package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"

	"github.com/schoentoon/go-cloud/pkg/models"

	"github.com/golang-jwt/jwt"
	"gorm.io/gorm"
)

// TODO make this all persistent
type Provider struct {
	DB         *gorm.DB
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

func NewProvider(db *gorm.DB) *Provider {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	return &Provider{
		DB:         db,
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func (p *Provider) Authenticate(user *models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"email": user.Email,
	})

	// Sign and get the complete encoded token as a string using the secret
	return token.SignedString(p.privateKey)
}

func (p *Provider) VerifyFromRequest(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}

		return p.publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		var user models.User
		result := p.DB.First(&user, "email = ?", claims["email"])

		if result.Error != nil {
			return nil, result.Error
		}
		return &user, nil
	} else {
		return nil, errors.New("Invalid token")
	}
}
