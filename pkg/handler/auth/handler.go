package auth

import (
	"github.com/gofiber/fiber/v3"

	authm "github.com/mymmrac/wof/pkg/module/auth"
	"github.com/mymmrac/wof/pkg/module/logger"
)

type handler struct {
	cfg  Config
	auth authm.Auth
}

func RegisterHandlers(cfg Config, router fiber.Router, auth authm.Auth) error {
	h := &handler{
		cfg:  cfg,
		auth: auth,
	}

	api := router.Group("/api")

	api.Post("/login", h.loginHandler)
	api.Post("/logout", authm.RequireMiddleware, h.logoutHandler)

	return nil
}

func (h *handler) loginHandler(fCtx fiber.Ctx) error {
	var request struct {
		Password string `json:"password"`
	}

	if err := fCtx.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest)
	}

	match, needsRehash, err := authm.ComparePasswordAndHash(request.Password, h.cfg.MasterPassword)
	if err != nil {
		logger.Errorw(fCtx, "compare password", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !match {
		return fiber.NewError(fiber.StatusUnauthorized)
	}
	if needsRehash {
		logger.Warnw(fCtx, "user needs to rehash password")
	}

	if err = h.auth.GenerateAndSetToken(fCtx); err != nil {
		logger.Errorw(fCtx, "set token cookie", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) logoutHandler(fCtx fiber.Ctx) error {
	h.auth.ClearToken(fCtx)
	return fCtx.Redirect().To("/")
}
