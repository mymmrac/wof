package item

import (
	"encoding/base64"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/mymmrac/wof/pkg/module/auth"
	"github.com/mymmrac/wof/pkg/module/id"
	"github.com/mymmrac/wof/pkg/module/item"
	"github.com/mymmrac/wof/pkg/module/logger"
)

type handler struct {
	itemRepository item.Repository
}

func RegisterHandlers(router fiber.Router, itemRepository item.Repository) {
	h := &handler{
		itemRepository: itemRepository,
	}

	api := router.Group("/api/wheel/:wheelID/item", auth.RequireMiddleware)

	api.Get("/", h.getAllHandler)
	api.Post("/", h.createHandler)
	api.Put("/order", h.updateItemOrderHandler)
	api.Put("/:itemID/rating", h.updateRatingHandler)
	api.Put("/:itemID/rejected", h.updateRejectedHandler)
	api.Put("/:itemID/used", h.updateUsedHandler)
	api.Put("/:itemID", h.updateHandler)
	api.Get("/:itemID/image", h.getImageHandler)
	api.Put("/:itemID/image", h.updateImageHandler)
	api.Delete("/:itemID", h.deleteHandler)
}

func (h *handler) getAllHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID `uri:"wheelID" validate:"required"`
	}

	if err := fCtx.Bind().URI(&request); err != nil {
		logger.Warnw(fCtx, "get items, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	models, err := h.itemRepository.GetByWheelID(fCtx, request.WheelID)
	if err != nil {
		logger.Errorw(fCtx, "get items", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	slices.SortFunc(models, func(a item.Model, b item.Model) int {
		return a.Order - b.Order
	})

	type itemInfo struct {
		ID       id.ID  `json:"id"`
		Name     string `json:"name,omitempty"`
		Rating   int    `json:"rating,omitempty"`
		Rejected bool   `json:"rejected,omitempty"`
		Used     bool   `json:"used,omitempty"`
	}

	response := make([]itemInfo, len(models))
	for i, model := range models {
		response[i] = itemInfo{
			ID:       model.ID,
			Name:     model.Name,
			Rating:   model.Rating,
			Rejected: model.Rejected,
			Used:     model.Used,
		}
	}

	return fCtx.JSON(response)
}

func (h *handler) createHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID  `uri:"wheelID" validate:"required"`
		Name    string `json:"name"   validate:"omitempty,min=1,max=64"`
		Image   string `json:"image"  validate:"-"`
	}

	if err := fCtx.Bind().All(&request); err != nil {
		logger.Warnw(fCtx, "create item, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	var imageData []byte
	if request.Image != "" {
		var err error
		imageData, err = base64.StdEncoding.DecodeString(request.Image)
		if err != nil {
			logger.Warnw(fCtx, "decode image", "error", err)
			return fiber.NewError(fiber.StatusBadRequest)
		}

		contentType := http.DetectContentType(imageData)
		if !strings.HasPrefix(contentType, "image/") {
			logger.Warnw(fCtx, "unexpected image content type", "type", contentType)
			return fiber.NewError(fiber.StatusBadRequest)
		}
	}

	count, err := h.itemRepository.CountByWheelID(fCtx, request.WheelID)
	if err != nil {
		logger.Errorw(fCtx, "get items count", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	request.Name = strings.TrimSpace(request.Name)

	itemID := id.New()
	now := time.Now()
	err = h.itemRepository.Create(fCtx, &item.Model{
		ID:        itemID,
		WheelID:   request.WheelID,
		Name:      request.Name,
		Order:     count,
		Rating:    0,
		Rejected:  false,
		Used:      false,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		logger.Errorw(fCtx, "create item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	if imageData != nil {
		if err = h.itemRepository.UpdateImage(fCtx, itemID, imageData); err != nil {
			logger.Errorw(fCtx, "update item image", "error", err)
			return fiber.NewError(fiber.StatusInternalServerError)
		}
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) updateHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID  `uri:"wheelID" validate:"required"`
		ID      id.ID  `uri:"itemID"  validate:"required"`
		Name    string `json:"name"  validate:"omitempty,min=1,max=64"`
	}

	if err := fCtx.Bind().All(&request); err != nil {
		logger.Warnw(fCtx, "update item, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	model, found, err := h.itemRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found || model.WheelID != request.WheelID {
		return fiber.NewError(fiber.StatusNotFound)
	}

	request.Name = strings.TrimSpace(request.Name)

	err = h.itemRepository.UpdateInfo(fCtx, request.ID, request.Name)
	if err != nil {
		logger.Errorw(fCtx, "update item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) updateRatingHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID `uri:"wheelID" validate:"required"`
		ID      id.ID `uri:"itemID"  validate:"required"`
		Rating  int   `json:"rating" validate:"min=0,max=10"`
	}

	if err := fCtx.Bind().All(&request); err != nil {
		logger.Warnw(fCtx, "update item rating, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	model, found, err := h.itemRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found || model.WheelID != request.WheelID {
		return fiber.NewError(fiber.StatusNotFound)
	}

	err = h.itemRepository.UpdateRating(fCtx, request.ID, request.Rating)
	if err != nil {
		logger.Errorw(fCtx, "update item rating", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) updateRejectedHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID  id.ID `uri:"wheelID" validate:"required"`
		ID       id.ID `uri:"itemID"  validate:"required"`
		Rejected bool  `json:"rejected"`
	}

	if err := fCtx.Bind().All(&request); err != nil {
		logger.Warnw(fCtx, "update item rejected, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	model, found, err := h.itemRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found || model.WheelID != request.WheelID {
		return fiber.NewError(fiber.StatusNotFound)
	}

	err = h.itemRepository.UpdateRejected(fCtx, request.ID, request.Rejected)
	if err != nil {
		logger.Errorw(fCtx, "update item rejected", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) updateUsedHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID `uri:"wheelID" validate:"required"`
		ID      id.ID `uri:"itemID"  validate:"required"`
		Used    bool  `json:"used"`
	}

	if err := fCtx.Bind().All(&request); err != nil {
		logger.Warnw(fCtx, "update item used, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	model, found, err := h.itemRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found || model.WheelID != request.WheelID {
		return fiber.NewError(fiber.StatusNotFound)
	}

	err = h.itemRepository.UpdateUsed(fCtx, request.ID, request.Used)
	if err != nil {
		logger.Errorw(fCtx, "update item used", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) updateItemOrderHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID   `uri:"wheelID" validate:"required"`
		IDs     []id.ID `json:"ids"    validate:"gt=0,dive,required"`
	}

	if err := fCtx.Bind().All(&request); err != nil {
		logger.Warnw(fCtx, "update item order, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	models, err := h.itemRepository.GetByWheelID(fCtx, request.WheelID)
	if err != nil {
		logger.Errorw(fCtx, "get items", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	if len(models) != len(request.IDs) {
		logger.Warnw(fCtx, "update item order, length doesn't match",
			"expected", len(models), "actual", len(request.IDs),
		)
		return fiber.NewError(fiber.StatusBadRequest)
	}
	for _, modelID := range request.IDs {
		if !slices.ContainsFunc(models, func(model item.Model) bool {
			return model.ID == modelID
		}) {
			logger.Warnw(fCtx, "update item order, unexpected item", "id", modelID)
			return fiber.NewError(fiber.StatusBadRequest)
		}
	}

	err = h.itemRepository.UpdateOrder(fCtx, request.IDs)
	if err != nil {
		logger.Errorw(fCtx, "update item order", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

const fallbackImage = `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#3f4f56">
    <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14zm-5.04-6.71l-2.75 3.54-1.96-2.36L6.5 17h11l-3.54-4.71z"/>
</svg>
`

func (h *handler) getImageHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID `uri:"wheelID" validate:"required"`
		ID      id.ID `uri:"itemID"  validate:"required"`
	}

	if err := fCtx.Bind().URI(&request); err != nil {
		logger.Warnw(fCtx, "get item image, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	model, found, err := h.itemRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found || model.WheelID != request.WheelID {
		return fiber.NewError(fiber.StatusNotFound)
	}

	imageData, found, err := h.itemRepository.GetImageByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get item image", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found {
		fCtx.Set(fiber.HeaderContentType, "image/svg+xml")
		return fCtx.Send([]byte(fallbackImage))
	}

	contentType := http.DetectContentType(imageData)
	fCtx.Set(fiber.HeaderContentType, contentType)

	return fCtx.Send(imageData)
}

func (h *handler) updateImageHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID   `uri:"wheelID" validate:"required"`
		ID      id.ID   `uri:"itemID"  validate:"required"`
		Image   *string `json:"image"  validate:"required_without=Remove,omitempty"`
		Remove  bool    `json:"remove"`
	}

	if err := fCtx.Bind().All(&request); err != nil {
		logger.Warnw(fCtx, "update item image, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	model, found, err := h.itemRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found || model.WheelID != request.WheelID {
		return fiber.NewError(fiber.StatusNotFound)
	}

	var imageData []byte
	if !request.Remove && request.Image != nil {
		imageData, err = base64.StdEncoding.DecodeString(*request.Image)
		if err != nil {
			logger.Warnw(fCtx, "decode image", "error", err)
			return fiber.NewError(fiber.StatusBadRequest)
		}

		contentType := http.DetectContentType(imageData)
		if !strings.HasPrefix(contentType, "image/") {
			logger.Warnw(fCtx, "unexpected image content type", "type", contentType)
			return fiber.NewError(fiber.StatusBadRequest)
		}
	}

	if err = h.itemRepository.UpdateImage(fCtx, request.ID, imageData); err != nil {
		logger.Errorw(fCtx, "update item image", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}

func (h *handler) deleteHandler(fCtx fiber.Ctx) error {
	var request struct {
		WheelID id.ID `uri:"wheelID" validate:"required"`
		ID      id.ID `uri:"itemID"  validate:"required"`
	}

	if err := fCtx.Bind().URI(&request); err != nil {
		logger.Warnw(fCtx, "delete item, bad request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest)
	}

	model, found, err := h.itemRepository.GetByID(fCtx, request.ID)
	if err != nil {
		logger.Errorw(fCtx, "get item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}
	if !found || model.WheelID != request.WheelID {
		return fiber.NewError(fiber.StatusNotFound)
	}

	if err = h.itemRepository.DeleteByID(fCtx, model.ID); err != nil {
		logger.Errorw(fCtx, "delete item", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return fCtx.JSON(fiber.Map{"ok": true})
}
