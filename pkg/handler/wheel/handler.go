package wheel

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/mymmrac/wof/pkg/module/auth"
	"github.com/mymmrac/wof/pkg/module/id"
	"github.com/mymmrac/wof/pkg/module/logger"
	"github.com/mymmrac/wof/pkg/module/wheel"
)

type handler struct {
	wheelRepository wheel.Repository
}

func RegisterHandlers(router fiber.Router, wheelRepository wheel.Repository) {
	h := &handler{
		wheelRepository: wheelRepository,
	}

	api := router.Group("/api/wheel", auth.RequireMiddleware)

	api.Get("/", h.getAllHandler)
	api.Post("/", h.createHandler)
	api.Get("/:wheelID", h.getHandler)
	api.Put("/:wheelID", h.updateHandler)
	api.Delete("/:wheelID", h.deleteHandler)
}

type wheelInfo struct {
	ID   id.ID  `json:"id"`
	Name string `json:"name"`
}

func (h *handler) getAllHandler(fCtx fiber.Ctx) error {
	models, err := h.wheelRepository.Get(fCtx)
	if err != nil {
		logger.Errorw(fCtx, "get wheels", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	response := make([]wheelInfo, len(models))
	for i, model := range models {
		response[i] = wheelInfo{
			ID:   model.ID,
			Name: model.Name,
		}
	}

	return fCtx.JSON(response)
}

func (h *handler) getHandler(fCtx fiber.Ctx) error {
	var request struct {
		ID id.ID `uri:"wheelID" validate:"required"`
	}

	if err := fCtx.Bind().URI(&request); err != nil {
		logger.Warnw(fCtx, "get wheel, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	model, found, err := h.wheelRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get wheel", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound)
	}

	return fCtx.JSON(&wheelInfo{
		ID:   model.ID,
		Name: model.Name,
	})
}

func (h *handler) createHandler(fCtx fiber.Ctx) error {
	var request struct {
		Name string `json:"name" validate:"min=1,max=64"`
	}

	if err := fCtx.Bind().Body(&request); err != nil {
		logger.Warnw(fCtx, "create wheel, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	request.Name = strings.TrimSpace(request.Name)

	now := time.Now()
	err := h.wheelRepository.Create(fCtx, &wheel.Model{
		ID:        id.New(),
		Name:      request.Name,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		logger.Errorw(fCtx, "create wheel", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) updateHandler(fCtx fiber.Ctx) error {
	var request struct {
		ID   id.ID  `uri:"wheelID" validate:"required"`
		Name string `json:"name"   validate:"min=1,max=64"`
	}

	if err := fCtx.Bind().All(&request); err != nil {
		logger.Warnw(fCtx, "update wheel, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	_, found, err := h.wheelRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get user", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound)
	}

	err = h.wheelRepository.UpdateName(fCtx, request.ID, request.Name)
	if err != nil {
		logger.Errorw(fCtx, "update wheel", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) deleteHandler(fCtx fiber.Ctx) error {
	var request struct {
		ID id.ID `uri:"wheelID" validate:"required"`
	}

	if err := fCtx.Bind().URI(&request); err != nil {
		logger.Warnw(fCtx, "delete wheel, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	_, found, err := h.wheelRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get user", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound)
	}

	if err = h.wheelRepository.DeleteByID(fCtx, request.ID); err != nil {
		logger.Errorw(fCtx, "delete wheel", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}
