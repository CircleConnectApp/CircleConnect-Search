package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitIndexes creates all required indexes for the search collection
func InitIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create text indexes for search
	textIndexModel := mongo.IndexModel{
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

	// Create prefix indexes for autocomplete
	titlePrefixIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "title", Value: 1}},
		Options: options.Index().SetName("title_index"),
	}

	tagsPrefixIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "tags", Value: 1}},
		Options: options.Index().SetName("tags_index"),
	}

	autocompletePrefixIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "autocomplete_phrases", Value: 1}},
		Options: options.Index().SetName("autocomplete_phrases_index"),
	}

	// Content type index for filtering
	contentTypeIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "content_type", Value: 1}},
		Options: options.Index().SetName("content_type_index"),
	}

	// Compound index for efficient filtering by content type and date
	dateContentTypeIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "content_type", Value: 1},
			{Key: "created_at", Value: -1},
		},
		Options: options.Index().SetName("content_type_date_index"),
	}

	// Create all indexes
	indexes := []mongo.IndexModel{
		textIndexModel,
		titlePrefixIndex,
		tagsPrefixIndex,
		autocompletePrefixIndex,
		contentTypeIndex,
		dateContentTypeIndex,
	}

	for _, index := range indexes {
		indexName, err := MongoDB.Collection("search_index").Indexes().CreateOne(ctx, index)
		if err != nil {
			log.Printf("Warning: Failed to create index %s: %v", indexName, err)
		} else {
			log.Printf("Created index: %s", indexName)
		}
	}

	log.Println("Finished initializing MongoDB indexes")
}
