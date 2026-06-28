package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/urfave/cli/v3"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload"

	"github.com/mymmrac/wof/pkg/handler/auth"
	"github.com/mymmrac/wof/pkg/handler/item"
	"github.com/mymmrac/wof/pkg/handler/static"
	"github.com/mymmrac/wof/pkg/handler/wheel"
	authm "github.com/mymmrac/wof/pkg/module/auth"
	itemm "github.com/mymmrac/wof/pkg/module/item"
	"github.com/mymmrac/wof/pkg/module/logger"
	wheelm "github.com/mymmrac/wof/pkg/module/wheel"
)

func main() {
	log, err := zap.NewProduction(zap.AddStacktrace(zap.DPanicLevel))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to initialize zap logger: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	logger.DefaultLogger.Store(log.Sugar())

	cmd := &cli.Command{
		Name:  "wof",
		Usage: "Wheel of Fortune",
		Commands: []*cli.Command{
			{
				Name: "serve",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "address",
						Usage:    "listen address",
						Sources:  cli.EnvVars("WOF_ADDRESS"),
						Required: true,
						Value:    ":8080",
						Aliases:  []string{"a"},
					},
					&cli.StringFlag{
						Name:     "jwt-secret",
						Usage:    "JWT secret",
						Sources:  cli.EnvVars("WOF_JWT_SECRET"),
						Required: true,
						Aliases:  []string{"j"},
					},
					&cli.StringFlag{
						Name:     "master-password",
						Usage:    "master password",
						Sources:  cli.EnvVars("WOF_MASTER_PASSWORD"),
						Required: true,
						Aliases:  []string{"m"},
					},
					&cli.StringFlag{
						Name:     "bolt-db",
						Usage:    "Bolt DB path",
						Sources:  cli.EnvVars("WOF_BOLT_DB"),
						Required: true,
						Aliases:  []string{"b"},
					},
				},
				Action: serve,
			},
			{
				Name: "hash-password",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "password",
						Usage:    "password to hash",
						Sources:  cli.EnvVars("WOF_PASSWORD"),
						Required: true,
						Aliases:  []string{"p"},
					},
				},
				Action: hashPassword,
			},
		},
	}

	if err = cmd.Run(ctx, os.Args); err != nil {
		logger.Errorw(ctx, "run command", "error", err)
		return
	}
}

func serve(ctx context.Context, cmd *cli.Command) error {
	views, err := static.LoadViews()
	if err != nil {
		return fmt.Errorf("load views: %w", err)
	}

	authHandler := authm.NewAuth(authm.Config{
		JWTSecret: cmd.String("jwt-secret"),
	})

	db, err := bolt.Open(cmd.String("bolt-db"), 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return fmt.Errorf("open bolt db: %w", err)
	}
	defer func() { _ = db.Close() }()

	wheelRepository := wheelm.NewRepository(db)
	itemRepository := itemm.NewRepository(db)

	v := validator.New(validator.WithRequiredStructEnabled())

	app := fiber.New(fiber.Config{
		Views:           views,
		AppName:         "wof",
		BodyLimit:       64 * 1024 * 1024,
		StructValidator: &FiberValidatorAdapter{v: v},
	})

	app.Use(authHandler.Middleware)

	if err = static.RegisterHandlers(app); err != nil {
		return fmt.Errorf("register static handlers: %w", err)
	}

	if err = auth.RegisterHandlers(auth.Config{
		MasterPassword: cmd.String("master-password"),
	}, app, authHandler); err != nil {
		return fmt.Errorf("register static handlers: %w", err)
	}

	wheel.RegisterHandlers(app, wheelRepository)
	item.RegisterHandlers(app, itemRepository)

	addr := cmd.String("address")
	logger.Infow(ctx, "starting server", "address", addr)
	if err = app.Listen(addr, fiber.ListenConfig{
		GracefulContext:       ctx,
		DisableStartupMessage: true,
	}); err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	return nil
}

func hashPassword(_ context.Context, cmd *cli.Command) error {
	hash, err := authm.HashPassword(cmd.String("password"))
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	fmt.Println(hash)
	return nil
}

type FiberValidatorAdapter struct {
	v *validator.Validate
}

func (v *FiberValidatorAdapter) Validate(value any) error {
	return v.v.Struct(value)
}
