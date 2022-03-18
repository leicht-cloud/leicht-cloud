package plugin

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt"
	"github.com/leicht-cloud/leicht-cloud/pkg/app"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
)

func (a *App) UnmarshalUserFromRequest(r *http.Request) (*models.User, error) {
	raw := r.Header.Get("X-Leicht-Cloud-User")
	if raw == "" {
		return nil, fmt.Errorf("X-Leicht-Cloud-User header missing")
	}

	// TODO: We want to get rid of ParseUnverified once we have a decent way of passing data from the host
	// to this container and we can pass along the jwt public key and can thus verify the user information.
	// at which point we will also want to uncomment the token.Valid below and actually include it in the if statement
	parser := jwt.Parser{}
	token, _, err := parser.ParseUnverified(raw, &app.UserClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*app.UserClaims); ok /* && token.Valid*/ {
		return &claims.User, nil
	} else {
		return nil, errors.New("Invalid token")
	}
}
