package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"ai-reader/internal/config"
	"ai-reader/internal/db"
	"ai-reader/internal/handler"
	"ai-reader/internal/service"
	"ai-reader/internal/worker"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

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

	bookSvc := service.NewBookService(database, cfg.StorageDir)
	pipeline := worker.NewPipeline(database, cfg)
	bookHandler := handler.NewBookHandler(bookSvc, pipeline)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.POST("/books/upload", bookHandler.Upload)
	e.GET("/books", bookHandler.List)
	e.GET("/books/:id/status", bookHandler.Status)
	e.DELETE("/books/:id", bookHandler.Delete)

	log.Printf("server starting on :%s", cfg.Port)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
