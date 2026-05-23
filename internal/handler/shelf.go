package handler

import (
	"net/http"

	"ai-reader/internal/service"

	"github.com/labstack/echo/v4"
)

type ShelfHandler struct {
	svc *service.BookService
}

func NewShelfHandler(svc *service.BookService) *ShelfHandler {
	return &ShelfHandler{svc: svc}
}

func (h *ShelfHandler) Create(c echo.Context) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&body); err != nil || body.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name required"})
	}
	shelf, err := h.svc.CreateShelf(body.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, shelf)
}

func (h *ShelfHandler) Delete(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := h.svc.DeleteShelf(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
