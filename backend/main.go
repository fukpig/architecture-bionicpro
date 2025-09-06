package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

type App struct {
	Config     *Config
	ClickHouse *ClickHouseClient
}

func main() {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize ClickHouse client
	clickhouseClient, err := NewClickHouseClient(config)
	if err != nil {
		log.Fatalf("Failed to initialize ClickHouse client: %v", err)
	}
	defer clickhouseClient.Close()

	// Create app instance
	app := &App{
		Config:     config,
		ClickHouse: clickhouseClient,
	}

	// Setup Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Reports endpoint with authentication
	router.GET("/reports", AuthMiddleware(config), app.GetReports)

	// Start server
	log.Printf("Starting server on port %s", config.Port)
	if err := router.Run(":" + config.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
