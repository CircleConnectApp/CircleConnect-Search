package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// User represents the user data extracted from the JWT token
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// AuthMiddleware handles user authentication using JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check if the header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Get the token
		tokenString := parts[1]

		// Get the JWT secret key from environment variables
		jwtSecret := os.Getenv("JWT_SECRET_KEY")
		if jwtSecret == "" {
			jwtSecret = "default_secret_key" // Only for development
		}

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			c.Abort()
			return
		}

		// Check if the token is valid
		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to extract token claims"})
			c.Abort()
			return
		}

		// Check token expiration
		exp, ok := claims["exp"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token expiration"})
			c.Abort()
			return
		}

		if time.Now().Unix() > int64(exp) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			c.Abort()
			return
		}

		// Extract user information
		user := User{
			ID:       claims["id"].(string),
			Username: claims["username"].(string),
			Email:    claims["email"].(string),
			Role:     claims["role"].(string),
		}

		// Set user in context
		c.Set("user", user)
		c.Next()
	}
}

// AdminMiddleware ensures that the user has admin privileges
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userValue, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
			c.Abort()
			return
		}

		user, ok := userValue.(User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract user from context"})
			c.Abort()
			return
		}

		// Check if user has admin role
		if user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ServiceAuthMiddleware authorizes internal service-to-service communication
func ServiceAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get service API key from header
		apiKey := c.GetHeader("X-Service-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Service API key required"})
			c.Abort()
			return
		}

		// Get expected API key from environment variables
		expectedAPIKey := os.Getenv("SERVICE_API_KEY")
		if expectedAPIKey == "" {
			expectedAPIKey = "default_service_key" // Only for development
		}

		// Validate API key
		if apiKey != expectedAPIKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid service API key"})
			c.Abort()
			return
		}

		c.Next()
	}
} 