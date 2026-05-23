package handler

import (
	"net/http"
	"os"
	"strconv"

	"aksara/internal/service"
	"aksara/internal/worker"

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

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file required"})
	}
	defer file.Close()

	book, err := h.svc.Upload(file, header, title)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	go h.pipeline.Process(book.ID)

	return c.JSON(http.StatusCreated, book)
}

func (h *BookHandler) List(c echo.Context) error {
	books, err := h.svc.List(0)
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

func (h *BookHandler) Retry(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	book, err := h.svc.GetByID(id)
	if err != nil || book == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	h.svc.ResetForRetry(id)
	go h.pipeline.Process(id)
	return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
}

func (h *BookHandler) MoveBook(c echo.Context) error {
	bookID, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var body struct {
		ShelfID int64 `json:"shelf_id"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if err := h.svc.MoveBook(bookID, body.ShelfID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
}

func (h *BookHandler) Cover(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	coverPath := h.svc.CoverPath(id)
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		return c.NoContent(http.StatusNotFound)
	}
	return c.File(coverPath)
}

func parseID(c echo.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}
