package data

import (
	"github.com/golang-jwt/jwt/v5"
)

type AuthzClaims struct {
	jwt.RegisteredClaims
	UserId string `json:"user_id"`
}
