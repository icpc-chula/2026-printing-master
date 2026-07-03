package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"printingmaster/internal/config"
	"printingmaster/internal/db"
	"printingmaster/internal/handler"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	printHandler := handler.NewPrintHandler(database, cfg.StorageDir)
	workerHandler := handler.NewWorkerHandler(database)

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Content-Type", "username", "teamname", "teamid", "location"},
	}))
	router.POST("/print", printHandler.HandlePrint)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.POST("/workers", workerHandler.CreateWorker)
	router.GET("/workers", workerHandler.ListWorkers)
	router.DELETE("/workers/:id", workerHandler.DeleteWorker)

	addr := ":" + cfg.Port
	log.Printf("printing master listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
