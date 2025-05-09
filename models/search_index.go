package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ContentType represents different types of content that can be indexed
type ContentType string

const (
	Post      ContentType = "post"
	Community ContentType = "community"
	User      ContentType = "user"
	Comment   ContentType = "comment"
)

// SearchIndex represents a document in the search index collection
type SearchIndex struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ContentID           string             `bson:"content_id" json:"content_id"`                               // Original ID of the content
	ContentType         ContentType        `bson:"content_type" json:"content_type"`                           // Type of content (post, community, user, comment)
	Title               string             `bson:"title,omitempty" json:"title"`                               // Title/name of the content
	Content             string             `bson:"content" json:"content"`                                     // Main content body
	Author              string             `bson:"author,omitempty" json:"author"`                             // Author username or ID
	Tags                []string           `bson:"tags,omitempty" json:"tags"`                                 // Associated tags or categories
	CreatedAt           time.Time          `bson:"created_at" json:"created_at"`                               // When the content was created
	UpdatedAt           time.Time          `bson:"updated_at" json:"updated_at"`                               // When the content was last updated
	IndexedAt           time.Time          `bson:"indexed_at" json:"indexed_at"`                               // When the content was indexed
	Score               float64            `bson:"score,omitempty" json:"score"`                               // For relevance scoring
	Metadata            map[string]any     `bson:"metadata,omitempty" json:"metadata"`                         // Additional metadata
	AutocompletePhrases []string           `bson:"autocomplete_phrases,omitempty" json:"autocomplete_phrases"` // Key phrases for autocomplete
	PopularityScore     float64            `bson:"popularity_score,omitempty" json:"popularity_score"`         // For ranking recommendations
}

// SearchResult represents the result of a search query
type SearchResult struct {
	ID          string      `json:"id"`
	ContentID   string      `json:"content_id"`
	ContentType ContentType `json:"content_type"`
	Title       string      `json:"title,omitempty"`
	Snippet     string      `json:"snippet"` // A preview/snippet of the content
	Author      string      `json:"author,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Score       float64     `json:"score"`
	Highlights  []string    `json:"highlights,omitempty"` // Highlighted parts that matched the query
}

// SearchQuery represents a search request
type SearchQuery struct {
	Query       string     `json:"query"`
	ContentType *string    `json:"content_type,omitempty"` // Filter by content type
	FromDate    *time.Time `json:"from_date,omitempty"`    // Filter by date range
	ToDate      *time.Time `json:"to_date,omitempty"`
	Author      string     `json:"author,omitempty"` // Filter by author
	Tags        []string   `json:"tags,omitempty"`   // Filter by tags
	Page        int        `json:"page"`             // For pagination
	PageSize    int        `json:"page_size"`
	SortBy      string     `json:"sort_by,omitempty"`    // Field to sort by
	SortOrder   string     `json:"sort_order,omitempty"` // asc or desc
}
