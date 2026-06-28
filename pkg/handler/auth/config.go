package auth

type Config struct {
	MasterPassword string `validate:"required"`
}
