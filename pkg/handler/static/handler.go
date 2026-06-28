package static

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"

	"github.com/mymmrac/wof/pkg/module/auth"
)

//go:embed views/*
var viewsFS embed.FS

//go:embed public/*
var publicFS embed.FS

type Views fiber.Views

func LoadViews() (Views, error) {
	viewsDirFR, err := fs.Sub(viewsFS, "views")
	if err != nil {
		return nil, fmt.Errorf("load views filesystem: %w", err)
	}

	views := html.NewFileSystem(http.FS(viewsDirFR), ".gohtml")
	return views, nil
}

func RegisterHandlers(router fiber.Router) error {
	router.Get("/", func(fCtx fiber.Ctx) error {
		_, ok := auth.UserFromContext(fCtx)
		if ok {
			return fCtx.Redirect().To("/home")
		}
		return fCtx.Render("index", nil, "layouts/main")
	})

	router.Get("/home", auth.RequireMiddleware, func(fCtx fiber.Ctx) error {
		return fCtx.Render("home", nil, "layouts/main")
	})

	router.Get("/wheel/:wheelID", auth.RequireMiddleware, func(fCtx fiber.Ctx) error {
		return fCtx.Render("wheel", fiber.Map{
			"WheelId": fCtx.Params("wheelID"),
		}, "layouts/main")
	})

	publicDirFR, err := fs.Sub(publicFS, "public")
	if err != nil {
		return fmt.Errorf("load public filesystem: %w", err)
	}

	router.Get("/*", static.New("", static.Config{
		FS:     publicDirFR,
		Browse: true,
	}))

	return nil
}
