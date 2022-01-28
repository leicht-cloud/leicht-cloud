package auth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"

	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/golang-jwt/jwt"
	"gorm.io/gorm"
)

// TODO make this all persistent
type Provider struct {
	DB         *gorm.DB
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

type Config struct {
	PrivateKey string `yaml:"private_key"`
}

func (c *Config) Create(db *gorm.DB) (*Provider, error) {
	if c.PrivateKey == "" {
		logrus.Warn("No key found to sign cookies with, generating one for you")
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}

		buf := bytes.Buffer{}
		err = yaml.NewEncoder(&buf).Encode(struct{ Auth Config }{Auth: Config{PrivateKey: string(privateKey)}})
		if err != nil {
			return nil, err
		}
		logrus.Infof("Consider adding the following to your configuration\n%s", buf.String())

		return &Provider{
			DB:         db,
			publicKey:  publicKey,
			privateKey: privateKey,
		}, nil
	}

	privateKey := ed25519.PrivateKey([]byte(c.PrivateKey))

	return &Provider{
		DB:         db,
		publicKey:  privateKey.Public().(ed25519.PublicKey),
		privateKey: privateKey,
	}, nil
}

type UserClaims struct {
	jwt.StandardClaims
	ID uint64
}

func (p *Provider) Authenticate(user *models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, UserClaims{
		ID: user.ID,
	})

	// Sign and get the complete encoded token as a string using the secret
	return token.SignedString(p.privateKey)
}

func (p *Provider) verifyCookie(cookie string) (*models.User, error) {
	token, err := jwt.ParseWithClaims(cookie, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}

		return p.publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		var user models.User
		result := p.DB.First(&user, claims.ID)

		if result.Error != nil {
			return nil, result.Error
		}
		return &user, nil
	} else {
		return nil, errors.New("Invalid token")
	}
}

func (p *Provider) VerifyFromRequest(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return nil, err
	}

	return p.verifyCookie(cookie.Value)
}
