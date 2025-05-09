package routes

import (
	"circleconnect-search/controllers"
	"circleconnect-search/middleware"

	"github.com/gin-gonic/gin"
)

// SearchController instance
var searchController = new(controllers.SearchController)

// SetupRoutes configures the API routes for the search service
func SetupRoutes(r *gin.Engine) {
	// Public routes
	search := r.Group("/api/search")
	{
		// Search endpoint - public access for basic searches
		search.GET("", searchController.Search)

		// Recommend endpoint - for autocomplete suggestions
		search.GET("/recommend", searchController.Recommend)

		// Trending endpoint - for popular search terms
		search.GET("/trending", searchController.TrendingSearches)
	}

	// Protected routes - require authentication
	protected := r.Group("/api/search")
	protected.Use(middleware.AuthMiddleware())
	{
		// Advanced search with filters - may be restricted based on user role
		protected.GET("/advanced", searchController.Search)
	}

	// Admin routes - for content indexing, only internal services should access these
	admin := r.Group("/api/search/admin")
	admin.Use(middleware.ServiceAuthMiddleware())
	{
		// Index a new document or update an existing one
		admin.POST("/index", searchController.Index)

		// Delete content from the index
		admin.DELETE("/index/:id", searchController.Delete)

		// Batch operations could be added here
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "search",
		})
	})
}
