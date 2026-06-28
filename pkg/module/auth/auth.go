package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

const tokenCookie = "token"

var userKey = userKeyType{} //nolint:gochecknoglobals

type userKeyType struct{}

type User struct {
	Authenticated bool
}

func UserFromContext(ctx context.Context) (User, bool) {
	authUser, ok := ctx.Value(userKey).(User)
	if !ok || !authUser.Authenticated {
		return User{}, false
	}
	return authUser, true
}

func MustUserFromContext(ctx context.Context) User {
	authUser, ok := UserFromContext(ctx)
	if !ok {
		panic(errors.New("user not found in context"))
	}
	return authUser
}

func RequireMiddleware(fCtx fiber.Ctx) error {
	_, ok := UserFromContext(fCtx)
	if ok {
		return fCtx.Next()
	}

	if fCtx.Method() == fiber.MethodGet && !strings.HasPrefix(fCtx.Path(), "/api/") {
		return fCtx.Redirect().To("/")
	}

	return fCtx.SendStatus(fiber.StatusUnauthorized)
}

type Auth interface {
	Middleware(fCtx fiber.Ctx) error
	GenerateAndSetToken(fCtx fiber.Ctx) error
	ClearToken(fCtx fiber.Ctx)
}

type auth struct {
	cfg Config
}

func NewAuth(cfg Config) Auth {
	return &auth{
		cfg: cfg,
	}
}

func (a *auth) Middleware(fCtx fiber.Ctx) error {
	token := fCtx.Cookies(tokenCookie)
	if token == "" {
		return fCtx.Next()
	}

	var claims jwt.RegisteredClaims
	parsedToken, err := jwt.ParseWithClaims(token, &claims, func(_ *jwt.Token) (any, error) {
		return []byte(a.cfg.JWTSecret), nil
	}, jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return fCtx.Next()
	}

	if !parsedToken.Valid {
		return fCtx.Next()
	}

	fCtx.Locals(userKey, User{
		Authenticated: true,
	})
	return fCtx.Next()
}

func (a *auth) GenerateAndSetToken(fCtx fiber.Ctx) error {
	expiresAt := time.Now().Add(time.Hour)
	token, err := a.generateJWT(expiresAt)
	if err != nil {
		return fmt.Errorf("generate JWT token: %w", err)
	}

	fCtx.Cookie(&fiber.Cookie{
		Name:     tokenCookie,
		Value:    token,
		HTTPOnly: true,
		Expires:  expiresAt,
	})

	return nil
}

func (a *auth) generateJWT(expiresAt time.Time) (string, error) {
	now := jwt.NewNumericDate(time.Now())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		NotBefore: now,
		IssuedAt:  now,
	})

	signedToken, err := token.SignedString([]byte(a.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signedToken, nil
}

func (a *auth) ClearToken(fCtx fiber.Ctx) {
	fCtx.Cookie(&fiber.Cookie{
		Name:     tokenCookie,
		HTTPOnly: true,
		Expires:  time.Now().Add(-time.Hour),
	})
}
