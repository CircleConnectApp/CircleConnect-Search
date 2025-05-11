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
	// Try to load environment variables from different files
	if err := godotenv.Load(); err != nil {
		// If .env fails, try dev.env
		if err := godotenv.Load("dev.env"); err != nil {
			log.Println("Warning: No .env or dev.env file found, using default values")
		} else {
			log.Println("Loaded configuration from dev.env")
		}
	} else {
		log.Println("Loaded configuration from .env")
	}

	// Check if we need to skip database connections (for development/testing)
	skipConnections := os.Getenv("SKIP_DB_INIT") == "true"

	var client *redis.Client

	// Only attempt Redis connection if not skipped
	if !skipConnections {
		// Get Redis URL from environment or use default
		redisURL := os.Getenv("REDIS_URL")
		if redisURL == "" {
			redisURL = "redis://localhost:6379"
			log.Println("REDIS_URL not set, using default:", redisURL)
		}

		// Parse Redis URL and connect
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("Error parsing Redis URL: %v. Using default connection.", err)
			// Fallback to direct connection
			client = redis.NewClient(&redis.Options{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			})
		} else {
			client = redis.NewClient(opt)
		}

		// Test Redis connection
		ping, err := client.Ping(ctx).Result()
		if err != nil {
			log.Println("Warning: Error connecting to Redis:", err.Error())
			log.Println("Continuing without Redis connection...")
		} else {
			fmt.Println("Connected to Redis:", ping)
		}

		// Make Redis client available to controllers
		controllers.RedisClient = client
	} else {
		log.Println("Skipping Redis connection (SKIP_DB_INIT=true)")
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

	// Initialize database connections if not skipped
	if !skipConnections {
		database.InitDBs()
	} else {
		log.Println("Skipping database initialization (SKIP_DB_INIT=true)")
	}

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

	// Set up API routes first
	routes.SetupRoutes(r)

	// Add health check endpoint separately from the main routes
	// This avoids conflicts with the gin router
	r.GET("/healthz", func(c *gin.Context) {
		// Check Redis connection if available
		redisStatus := "skipped"
		if client != nil {
			_, err := client.Ping(ctx).Result()
			if err != nil {
				redisStatus = "error: " + err.Error()
			} else {
				redisStatus = "ok"
			}
		}

		c.JSON(200, gin.H{
			"status":       "ok",
			"service":      "user-search-service",
			"redis_status": redisStatus,
			"env":          os.Getenv("ENVIRONMENT"),
		})
	})

	// Add a test route for development (when DBs are skipped)
	if skipConnections {
		r.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Search service is running in development mode",
				"status":  "ok",
			})
		})
	}

	// Start server
	log.Printf("Search Service starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
