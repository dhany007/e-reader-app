package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"aksara/internal/config"
	"aksara/internal/db"
	"aksara/internal/handler"
	appMiddleware "aksara/internal/middleware"
	"aksara/internal/model"
	"aksara/internal/service"
	"aksara/internal/worker"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type templateRenderer struct {
	tmpl *template.Template
}

func (t *templateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(cfg.StorageDir, "pdfs"), 0755); err != nil {
		log.Fatalf("create storage dirs: %v", err)
	}

	database, err := db.Init(cfg.DataDir)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	funcMap := template.FuncMap{
		"progressPct": func(done, total int) int {
			if total == 0 {
				return 0
			}
			return done * 100 / total
		},
		"statusIs": func(status model.BookStatus, s string) bool {
			return string(status) == s
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html"))

	bookSvc := service.NewBookService(database, cfg.StorageDir)
	pipeline := worker.NewPipeline(database, cfg)
	bookHandler := handler.NewBookHandler(bookSvc, pipeline)
	authHandler := handler.NewAuthHandler(cfg)
	readerHandler := handler.NewReaderHandler(bookSvc)
	shelfHandler := handler.NewShelfHandler(bookSvc)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Renderer = &templateRenderer{tmpl: tmpl}

	// Public routes
	e.GET("/", func(c echo.Context) error { return c.Redirect(http.StatusFound, "/library") })
	e.GET("/login", authHandler.LoginPage)
	e.POST("/login", authHandler.Login)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Protected routes
	g := e.Group("", appMiddleware.Auth(cfg.SessionSecret))
	g.GET("/logout", authHandler.Logout)
	g.GET("/library", readerHandler.Library)
	g.POST("/books/upload", bookHandler.Upload)
	g.GET("/books", bookHandler.List)
	g.GET("/books/:id/status", bookHandler.Status)
	g.DELETE("/books/:id", bookHandler.Delete)
	g.GET("/books/:id/read", readerHandler.Read)
	g.GET("/books/:id/pages/:num", readerHandler.GetPage)
	g.POST("/books/:id/progress", readerHandler.SaveProgress)
	g.GET("/books/:id/progress", readerHandler.GetProgress)
	g.POST("/books/:id/shelf", bookHandler.MoveBook)
	g.POST("/shelves", shelfHandler.Create)
	g.DELETE("/shelves/:id", shelfHandler.Delete)

	log.Printf("server starting on :%s", cfg.Port)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
