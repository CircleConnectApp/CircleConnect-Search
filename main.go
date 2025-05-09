package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"circleconnect-search/controllers"
	"circleconnect-search/database"
	"circleconnect-search/routes"
)

var ctx = context.Background()

func main() {
	// Initialize Redis for caching
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Test Redis connection
	ping, err := client.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Error connecting to Redis:", err.Error())
		return
	}
	fmt.Println("Connected to Redis:", ping)

	// Make Redis client available to controllers
	controllers.RedisClient = client

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Error loading .env file, using default values")
	}

	// Get JWT secret
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Println("Warning: JWT_SECRET_KEY not set in environment, using default value")
		jwtSecret = "default_secret_key" // Only for development
	}
	log.Println("JWT secret loaded successfully")

	// Get port from environment variables or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	// Initialize database connections
	database.InitDBs()

	// Create Gin router
	r := gin.Default()

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Set up routes
	routes.SetupRoutes(r)

	// Start server
	log.Printf("Search Service starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
