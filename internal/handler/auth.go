package handler

import (
	"net/http"

	"ai-reader/internal/config"
	appMiddleware "ai-reader/internal/middleware"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

func (h *AuthHandler) LoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{
		"error": c.QueryParam("error"),
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username != h.cfg.AdminUsername {
		return c.Redirect(http.StatusFound, "/login?error=1")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(h.cfg.AdminPasswordHash), []byte(password)); err != nil {
		return c.Redirect(http.StatusFound, "/login?error=1")
	}

	appMiddleware.SetSession(c, username, h.cfg.SessionSecret)
	return c.Redirect(http.StatusFound, "/library")
}

func (h *AuthHandler) Logout(c echo.Context) error {
	appMiddleware.ClearSession(c)
	return c.Redirect(http.StatusFound, "/login")
}
