package handler

import (
	"net/http"
	"strconv"

	"aksara/internal/service"

	"github.com/labstack/echo/v4"
)

const shelfUncategorized = int64(-1)

type ReaderHandler struct {
	svc *service.BookService
}

func NewReaderHandler(svc *service.BookService) *ReaderHandler {
	return &ReaderHandler{svc: svc}
}

func (h *ReaderHandler) Library(c echo.Context) error {
	// Parse shelf filter: 0=all, -1=uncategorized, N=specific shelf
	shelfID := int64(0)
	if s := c.QueryParam("shelf"); s != "" {
		shelfID, _ = strconv.ParseInt(s, 10, 64)
	}

	books, err := h.svc.List(shelfID)
	if err != nil {
		return err
	}
	shelves, err := h.svc.ListShelves()
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "library.html", map[string]interface{}{
		"books":         books,
		"shelves":       shelves,
		"activeShelfID": shelfID,
	})
}

func (h *ReaderHandler) Read(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	book, err := h.svc.GetByID(id)
	if err != nil || book == nil {
		return c.Redirect(http.StatusFound, "/library")
	}
	return c.Render(http.StatusOK, "reader.html", map[string]interface{}{
		"book": book,
	})
}

func (h *ReaderHandler) GetPage(c echo.Context) error {
	bookID, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	pageNum, err := strconv.Atoi(c.Param("num"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid page"})
	}
	page, err := h.svc.GetPage(bookID, pageNum)
	if err != nil || page == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"html": page.HTMLContent,
	})
}

func (h *ReaderHandler) SaveProgress(c echo.Context) error {
	bookID, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var body struct {
		ScrollPct float64 `json:"scroll_pct"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if err := h.svc.SaveProgress(bookID, body.ScrollPct); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
}

func (h *ReaderHandler) GetProgress(c echo.Context) error {
	bookID, err := parseID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	pct, err := h.svc.GetProgress(bookID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"scroll_pct": pct,
	})
}
