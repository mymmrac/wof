package auth

type Config struct {
	JWTSecret string `validate:"required"`
}
