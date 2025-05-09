# CircleConnect User Search Service

This service provides user search functionality for the CircleConnect platform using Redis for caching.

## Dockerizing the User Search Service

### Prerequisites
- Docker and Docker Compose
- Go 1.21+

### Configuration

1. Make sure your Go code connects to Redis using the environment variable:

```go
// Example Redis connection in your Go code
func connectToRedis() (*redis.Client, error) {
    redisURL := os.Getenv("REDIS_URL")
    if redisURL == "" {
        redisURL = "redis://localhost:6379"
    }
    
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }
    
    client := redis.NewClient(opt)
    _, err = client.Ping(context.Background()).Result()
    return client, err
}
```

2. Update any hard-coded Redis connection strings to use the environment variable instead.

### Running with Docker

1. Make sure you have the `docker-compose.yml` and `Dockerfile` in your project directory.

2. Build and run the containers:
```bash
docker-compose build
docker-compose up -d
```

3. To stop the containers:
```bash
docker-compose down
```

### Troubleshooting Redis Connection

If you see the error: `Error connecting to Redis: dial tcp [::1]:6379: connectex: No connection could be made because the target machine actively refused it`, it means:

1. You're trying to connect to Redis on localhost ([::1]:6379) instead of the Redis container
2. You need to update the connection string to use the environment variable `REDIS_URL`

### Environment Variables

The following environment variables are available:
- `PORT` - Server port (default: 4005)
- `ENVIRONMENT` - Runtime environment (development/production)
- `JWT_SECRET` - Secret key for JWT validation
- `USER_SERVICE_URL` - User service endpoint URL
- `REDIS_URL` - Redis connection string

### Testing the Connection

To test if your service can connect to Redis in Docker:

1. Run the service with Docker Compose
2. Check the logs for any Redis connection errors:
```bash
docker-compose logs user-search-service
```

3. Connect to the Redis container directly:
```bash
docker-compose exec redis redis-cli
```

## Features

- Content search across multiple sources (Communities, Posts, Users, Comments)
- Real-time search suggestions and autocompletion
- Trending search terms and hashtags
- Full-text search capabilities
- Filtering by content type, date, author, and tags
- Result caching for improved performance
- Integration with MongoDB for storing search indexes
- Integration with PostgreSQL for accessing relational data
- Service-to-service authentication

## Setup and Installation

### Prerequisites

- Go 1.20 or later
- MongoDB
- PostgreSQL
- Redis (for caching)

### Environment Variables

Create a `.env` file in the root directory with the following variables:

```
# Server configuration
PORT=8084

# PostgreSQL configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=yehia
POSTGRES_DB=circleConnect

# MongoDB configuration
MONGO_URI=mongodb://localhost:27017
MONGO_DB=circleconnect_search

# Redis configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Security
JWT_SECRET_KEY=your_jwt_secret_key
SERVICE_API_KEY=your_service_api_key
```

### Running the Service

```bash
# Download dependencies
go mod download

# Build the service
go build -o search-service

# Run the service
./search-service
```

## API Endpoints

### Public Endpoints

- `GET /api/search?q={query}&type={contentType}&page={page}&size={size}`
  - Search for content with query and optional filters
  - Parameters:
    - `q`: Search query (required)
    - `type`: Content type (post, community, user, comment)
    - `page`: Page number (default: 1)
    - `size`: Results per page (default: 10, max: 50)

- `GET /api/search/recommend?prefix={prefix}&type={contentType}`
  - Get real-time autocomplete suggestions as the user types
  - Parameters:
    - `prefix`: The characters that the user has typed (required)
    - `type`: Content type filter (optional)
  - Returns a list of suggested search terms based on the prefix

- `GET /api/search/trending?type={contentType}&limit={limit}`
  - Get trending/popular search terms across the platform
  - Parameters:
    - `type`: Content type filter (optional)
    - `limit`: Maximum number of results (default: 10, max: 50)
  - Returns popular search terms ranked by popularity

### Protected Endpoints (require authentication)

- `GET /api/search/advanced?q={query}&...`
  - Advanced search with additional filters
  - Same parameters as the public endpoint, plus:
    - `from_date`: Filter results from this date
    - `to_date`: Filter results until this date
    - `author`: Filter by author
    - `tags`: Filter by tags (comma-separated)

### Admin Endpoints (for internal service usage)

- `POST /api/search/admin/index`
  - Index or update content in the search index
  - Requires a service API key in the `X-Service-API-Key` header
  - Body: JSON object with content details

- `DELETE /api/search/admin/index/{id}?type={contentType}`
  - Remove content from the search index
  - Requires a service API key in the `X-Service-API-Key` header

## Architecture

The Search Service uses a combination of MongoDB and PostgreSQL:
- MongoDB is used for storing and querying the search indexes
- PostgreSQL is used for accessing relational data

Redis is used for caching search results to improve performance for repeated queries.

## Integration with Other Services

Other services can interact with the Search Service in two ways:
1. By calling the search API to retrieve search results
2. By sending data to be indexed through the admin API

Example service-to-service communication for indexing new content:

```go
// When a new post is created in the Post Service
func indexNewPost(post *models.Post) error {
    searchIndex := map[string]interface{}{
        "content_id":   post.ID,
        "content_type": "post",
        "title":        post.Title,
        "content":      post.Content,
        "author":       post.AuthorID,
        "created_at":   post.CreatedAt,
        "updated_at":   post.UpdatedAt,
        "tags":         post.Tags,
    }
    
    jsonData, err := json.Marshal(searchIndex)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequest("POST", "http://search-service:8084/api/search/admin/index", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Service-API-Key", os.Getenv("SERVICE_API_KEY"))
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("failed to index post, status: %d", resp.StatusCode)
    }
    
    return nil
}
``` 