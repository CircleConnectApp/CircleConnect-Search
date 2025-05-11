package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"circleconnect-search/database"
	"circleconnect-search/models"
)

// RedisClient is used for caching search results
var RedisClient *redis.Client

// Default values for pagination
const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 50
	maxSuggestions  = 10
)

// SearchController handles search operations
type SearchController struct{}

// Search handles search requests
func (sc *SearchController) Search(c *gin.Context) {
	// Extract search query parameters
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	// Parse pagination parameters
	page, pageSize := getPaginationParams(c)

	// Parse content type filter
	contentType := c.Query("type")

	// Try to get cached results
	cacheKey := fmt.Sprintf("search:%s:%s:%d:%d", query, contentType, page, pageSize)
	cachedResults, err := getCachedResults(cacheKey)
	if err == nil {
		c.JSON(http.StatusOK, cachedResults)
		return
	}

	// Prepare search query
	filter := bson.M{
		"$text": bson.M{
			"$search": query,
		},
	}

	// Add content type filter if specified
	if contentType != "" {
		filter["content_type"] = contentType
	}

	// Setup options for pagination and sorting
	findOptions := options.Find()
	findOptions.SetLimit(int64(pageSize))
	findOptions.SetSkip(int64((page - 1) * pageSize))
	findOptions.SetSort(bson.M{"score": -1}) // Sort by relevance
	findOptions.SetProjection(bson.M{
		"score": bson.M{"$meta": "textScore"},
	})

	// Execute search query
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := database.MongoDB.Collection("search_index").Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Search error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute search"})
		return
	}
	defer cursor.Close(ctx)

	// Process results
	var results []models.SearchResult
	for cursor.Next(ctx) {
		var document models.SearchIndex
		if err := cursor.Decode(&document); err != nil {
			log.Printf("Error decoding search result: %v", err)
			continue
		}

		// Convert to search result
		result := models.SearchResult{
			ID:          document.ID.Hex(),
			ContentID:   document.ContentID,
			ContentType: document.ContentType,
			Title:       document.Title,
			Snippet:     createSnippet(document.Content, query),
			Author:      document.Author,
			CreatedAt:   document.CreatedAt,
			UpdatedAt:   document.UpdatedAt,
			Score:       document.Score,
		}

		results = append(results, result)
	}

	// Cache results
	responseData := gin.H{
		"results": results,
		"page":    page,
		"size":    pageSize,
		"total":   len(results),
		"query":   query,
	}

	cacheResults(cacheKey, responseData)

	c.JSON(http.StatusOK, responseData)
}

// Index handles indexing new content
func (sc *SearchController) Index(c *gin.Context) {
	var indexRequest models.SearchIndex
	if err := c.ShouldBindJSON(&indexRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set indexed time
	indexRequest.IndexedAt = time.Now()

	// If the ID is not set, generate a new one
	if indexRequest.ID.IsZero() {
		indexRequest.ID = primitive.NewObjectID()
	}

	// Extract key phrases for autocomplete if not provided
	if len(indexRequest.AutocompletePhrases) == 0 {
		indexRequest.AutocompletePhrases = extractKeyPhrases(indexRequest)
	}

	// Set default popularity score if not provided
	if indexRequest.PopularityScore == 0 {
		indexRequest.PopularityScore = 1.0 // Default score
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Upsert the document
	filter := bson.M{"content_id": indexRequest.ContentID, "content_type": indexRequest.ContentType}
	update := bson.M{"$set": indexRequest}
	opts := options.Update().SetUpsert(true)

	result, err := database.MongoDB.Collection("search_index").UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Printf("Indexing error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to index content"})
		return
	}

	// Create text index on collection if it doesn't exist
	ensureTextIndex(ctx)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Content indexed successfully",
		"upserted_id": indexRequest.ID.Hex(),
		"matched":     result.MatchedCount,
		"modified":    result.ModifiedCount,
		"upserted":    result.UpsertedCount,
	})
}

// extractKeyPhrases extracts important phrases from content for autocomplete
func extractKeyPhrases(doc models.SearchIndex) []string {
	phrases := make(map[string]bool)

	// Add title if available
	if doc.Title != "" {
		phrases[doc.Title] = true

		// Add title words that are significant (length > 3)
		words := strings.Fields(doc.Title)
		for _, word := range words {
			if len(word) > 3 {
				phrases[word] = true
			}
		}
	}

	// Add all tags
	for _, tag := range doc.Tags {
		phrases[tag] = true
	}

	// Extract significant phrases from content
	if doc.Content != "" {
		// Simple approach: split by spaces and punctuation
		words := strings.FieldsFunc(doc.Content, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})

		// Add significant words (length > 3)
		for _, word := range words {
			if len(word) > 3 {
				phrases[word] = true
			}
		}

		// TODO: More sophisticated phrase extraction could be added here
		// For example, extracting n-grams or using NLP techniques
	}

	// Convert map to slice
	result := make([]string, 0, len(phrases))
	for phrase := range phrases {
		result = append(result, phrase)
	}

	return result
}

// ensureTextIndex ensures that text indexes exist on the necessary fields
func ensureTextIndex(ctx context.Context) {
	// Define the text index model
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "title", Value: "text"},
			{Key: "content", Value: "text"},
			{Key: "tags", Value: "text"},
			{Key: "autocomplete_phrases", Value: "text"},
		},
		Options: options.Index().SetWeights(bson.D{
			{Key: "title", Value: 10},
			{Key: "tags", Value: 5},
			{Key: "autocomplete_phrases", Value: 3},
			{Key: "content", Value: 1},
		}).SetName("text_search_index"),
	}

	// Create the index
	_, err := database.MongoDB.Collection("search_index").Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Warning: Failed to create text index: %v", err)
	}
}

// Delete removes content from the search index
func (sc *SearchController) Delete(c *gin.Context) {
	contentID := c.Param("id")
	contentType := c.Query("type")

	if contentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content ID is required"})
		return
	}

	filter := bson.M{"content_id": contentID}
	if contentType != "" {
		filter["content_type"] = contentType
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := database.MongoDB.Collection("search_index").DeleteMany(ctx, filter)
	if err != nil {
		log.Printf("Delete error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete content from index"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Content removed from index successfully",
		"deleted_count": result.DeletedCount,
	})
}

// Recommend provides real-time search suggestions as the user types
func (sc *SearchController) Recommend(c *gin.Context) {
	// Get the user input (prefix)
	prefix := c.Query("prefix")
	if prefix == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Prefix parameter is required"})
		return
	}

	// Get content type if specified (optional filter)
	contentType := c.Query("type")

	// Try to get cached suggestions
	cacheKey := fmt.Sprintf("suggestions:%s:%s", prefix, contentType)
	cachedSuggestions, err := getCachedResults(cacheKey)
	if err == nil {
		c.JSON(http.StatusOK, cachedSuggestions)
		return
	}

	// Create the context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Prepare the filter for MongoDB
	filter := bson.M{
		"$or": []bson.M{
			// Search for title starting with the prefix (case insensitive)
			{"title": bson.M{"$regex": "^" + regexp.QuoteMeta(prefix), "$options": "i"}},
			// Search for content words starting with the prefix
			{"content": bson.M{"$regex": "\\b" + regexp.QuoteMeta(prefix), "$options": "i"}},
			// Search in tags
			{"tags": bson.M{"$regex": "^" + regexp.QuoteMeta(prefix), "$options": "i"}},
		},
	}

	// Add content type filter if specified
	if contentType != "" {
		filter["content_type"] = contentType
	}

	// Set options for MongoDB query
	findOptions := options.Find()
	findOptions.SetLimit(int64(maxSuggestions))

	// Execute the query
	cursor, err := database.MongoDB.Collection("search_index").Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Suggest error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suggestions"})
		return
	}
	defer cursor.Close(ctx)

	// Process suggestions
	var suggestions []string
	termMap := make(map[string]bool) // To avoid duplicates

	// Process the cursor results
	for cursor.Next(ctx) {
		var document models.SearchIndex
		if err := cursor.Decode(&document); err != nil {
			log.Printf("Error decoding suggestion result: %v", err)
			continue
		}

		// Add title suggestions if title starts with prefix
		if document.Title != "" && strings.HasPrefix(strings.ToLower(document.Title), strings.ToLower(prefix)) {
			if !termMap[document.Title] {
				suggestions = append(suggestions, document.Title)
				termMap[document.Title] = true
			}
		}

		// Extract words from content that start with the prefix
		if document.Content != "" {
			words := strings.Fields(document.Content)
			for _, word := range words {
				if len(word) > 2 && strings.HasPrefix(strings.ToLower(word), strings.ToLower(prefix)) {
					if !termMap[word] {
						suggestions = append(suggestions, word)
						termMap[word] = true
					}
				}
			}
		}

		// Add matching tags
		for _, tag := range document.Tags {
			if strings.HasPrefix(strings.ToLower(tag), strings.ToLower(prefix)) {
				if !termMap[tag] {
					suggestions = append(suggestions, tag)
					termMap[tag] = true
				}
			}
		}

		// Limit to max suggestions
		if len(suggestions) >= maxSuggestions {
			break
		}
	}

	// Prepare response
	responseData := gin.H{
		"suggestions": suggestions,
		"prefix":      prefix,
	}

	// Cache the results for 5 minutes
	cacheResults(cacheKey, responseData)

	c.JSON(http.StatusOK, responseData)
}

// TrendingSearches returns the most popular search terms
func (sc *SearchController) TrendingSearches(c *gin.Context) {
	// Get content type if specified (optional filter)
	contentType := c.Query("type")

	// Get the limit parameter (default to 10)
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Try to get cached trending terms
	cacheKey := fmt.Sprintf("trending:%s:%d", contentType, limit)
	cachedTrending, err := getCachedResults(cacheKey)
	if err == nil {
		c.JSON(http.StatusOK, cachedTrending)
		return
	}

	// Create the context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Prepare the aggregation pipeline
	pipeline := []bson.M{
		{
			"$project": bson.M{
				"phrases":          "$autocomplete_phrases",
				"content_type":     1,
				"popularity_score": 1,
			},
		},
	}

	// Add content type filter if specified
	if contentType != "" {
		pipeline = append(pipeline, bson.M{
			"$match": bson.M{
				"content_type": contentType,
			},
		})
	}

	// Unwind the phrases array to get individual phrases
	pipeline = append(pipeline, bson.M{
		"$unwind": "$phrases",
	})

	// Group by phrase and sum popularity scores
	pipeline = append(pipeline, bson.M{
		"$group": bson.M{
			"_id":   "$phrases",
			"score": bson.M{"$sum": "$popularity_score"},
			"count": bson.M{"$sum": 1},
		},
	})

	// Sort by score (descending)
	pipeline = append(pipeline, bson.M{
		"$sort": bson.M{
			"score": -1,
			"count": -1,
		},
	})

	// Limit the results
	pipeline = append(pipeline, bson.M{
		"$limit": limit,
	})

	// Execute the aggregation
	cursor, err := database.MongoDB.Collection("search_index").Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Trending error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trending terms"})
		return
	}
	defer cursor.Close(ctx)

	// Process the results
	type TrendingTerm struct {
		Term  string  `bson:"_id" json:"term"`
		Score float64 `bson:"score" json:"score"`
		Count int     `bson:"count" json:"count"`
	}

	var trendingTerms []TrendingTerm
	if err := cursor.All(ctx, &trendingTerms); err != nil {
		log.Printf("Error processing trending terms: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process trending terms"})
		return
	}

	// Prepare the response
	responseData := gin.H{
		"trending": trendingTerms,
		"count":    len(trendingTerms),
	}

	// Cache the results for 1 hour (trending changes less frequently)
	cacheResults(cacheKey, responseData)

	c.JSON(http.StatusOK, responseData)
}

// Utility functions

// getPaginationParams extracts pagination parameters from the request
func getPaginationParams(c *gin.Context) (int, int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(defaultPage)))
	if err != nil || page < 1 {
		page = defaultPage
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defaultPageSize)))
	if err != nil || pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

// createSnippet generates a short snippet from the content
func createSnippet(content string, query string) string {
	// Simple implementation - in a real-world scenario, this would be more sophisticated
	maxLength := 150
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}

// getCachedResults attempts to retrieve search results from Redis cache
func getCachedResults(key string) (gin.H, error) {
	if RedisClient == nil {
		log.Println("Redis client not available, skipping cache lookup")
		return nil, fmt.Errorf("Redis client not initialized")
	}

	ctx := context.Background()
	cachedData, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Redis cache lookup error: %v", err)
		}
		return nil, err
	}

	var results gin.H
	if err := json.Unmarshal([]byte(cachedData), &results); err != nil {
		log.Printf("Error unmarshaling cached data: %v", err)
		return nil, err
	}

	return results, nil
}

// cacheResults stores search results in Redis with a TTL
func cacheResults(key string, results gin.H) {
	if RedisClient == nil {
		log.Println("Redis client not available, skipping cache storage")
		return
	}

	ctx := context.Background()
	jsonData, err := json.Marshal(results)
	if err != nil {
		log.Printf("Error marshaling results for cache: %v", err)
		return
	}

	// Cache results for 10 minutes
	err = RedisClient.Set(ctx, key, jsonData, 10*time.Minute).Err()
	if err != nil {
		log.Printf("Error caching search results: %v", err)
	}
}
