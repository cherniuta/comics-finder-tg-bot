package aaa

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"yadro.com/course/api/core"
)

const secretKey = "something secret here" // token sign key
const adminRole = "superuser"             // token subject

// Authentication, Authorization, Accounting
type AAA struct {
	users    map[string]string
	tokenTTL time.Duration
	log      *slog.Logger
}

func New(tokenTTL time.Duration, log *slog.Logger) (AAA, error) {
	const adminUser = "ADMIN_USER"
	const adminPass = "ADMIN_PASSWORD"
	user, ok := os.LookupEnv(adminUser)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin user from enviroment")
	}
	password, ok := os.LookupEnv(adminPass)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin password from enviroment")
	}

	return AAA{
		users:    map[string]string{user: password},
		tokenTTL: tokenTTL,
		log:      log,
	}, nil
}

func (a AAA) Login(name, password string) (string, error) {

	_, exists := a.users[name]
	if !exists {
		return "", core.ErrBadCredentials
	}
	if a.users[name] != password {
		return "", core.ErrBadCredentials
	}

	payload := jwt.MapClaims{
		"sub":  adminRole,
		"name": name,
		"exp":  jwt.NewNumericDate(time.Now().Add(a.tokenTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	t, err := token.SignedString([]byte(secretKey))
	if err != nil {
		a.log.Error("JWT token signing")
		return "", err
	}

	return t, nil
}

func (a AAA) Verify(tokenString string) error {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return core.ErrBadCredentials
	}
	if !token.Valid {
		return core.ErrBadCredentials
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		exp, ok := claims["exp"].(float64)
		if !ok {
			return core.ErrBadCredentials
		}

		if time.Now().Unix() > int64(exp) {
			return core.ErrBadCredentials
		}
	}

	return nil
}
