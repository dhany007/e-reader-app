package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const cookieName = "session"

func Auth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(cookieName)
			if err != nil || !validToken(cookie.Value, secret) {
				return c.Redirect(http.StatusFound, "/login")
			}
			return next(c)
		}
	}
}

func SetSession(c echo.Context, username, secret string) {
	token := username + "." + sign(username, secret)
	cookie := new(http.Cookie)
	cookie.Name = cookieName
	cookie.Value = token
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)
}

func ClearSession(c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = cookieName
	cookie.Value = ""
	cookie.Path = "/"
	cookie.MaxAge = -1
	c.SetCookie(cookie)
}

func validToken(token, secret string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	expected := sign(parts[0], secret)
	return hmac.Equal([]byte(parts[1]), []byte(expected))
}

func sign(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
