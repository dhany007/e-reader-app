package handler

import (
	"net/http"
	"strconv"

	"ai-reader/internal/service"
	"ai-reader/internal/worker"

	"github.com/labstack/echo/v4"
)

type BookHandler struct {
	svc      *service.BookService
	pipeline *worker.Pipeline
}

func NewBookHandler(svc *service.BookService, pipeline *worker.Pipeline) *BookHandler {
	return &BookHandler{svc: svc, pipeline: pipeline}
}

func (h *BookHandler) Upload(c echo.Context) error {
	title := c.FormValue("title")
	category := c.FormValue("category")

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file required"})
	}
	defer file.Close()

	book, err := h.svc.Upload(file, header, title, category)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	go h.pipeline.Process(book.ID)

	return c.JSON(http.StatusCreated, book)
}

func (h *BookHandler) List(c echo.Context) error {
	books, err := h.svc.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, books)
}

func (h *BookHandler) Status(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	book, err := h.svc.GetByID(id)
	if err != nil || book == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":      book.Status,
		"total_pages": book.TotalPages,
		"done_pages":  book.DonePages,
	})
}

func (h *BookHandler) Delete(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	if err := h.svc.Delete(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func parseID(c echo.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}
