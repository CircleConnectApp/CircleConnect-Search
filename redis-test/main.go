package main

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Get Redis URL from environment or use default
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
		fmt.Println("REDIS_URL not set, using default:", redisURL)
	}

	fmt.Println("Attempting to connect to Redis at:", redisURL)

	// Parse Redis URL and connect
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		fmt.Printf("Error parsing Redis URL: %v\n", err)
		return
	}

	client := redis.NewClient(opt)
	defer client.Close()

	// Test Redis connection
	ctx := context.Background()
	ping, err := client.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Error connecting to Redis:", err.Error())
		return
	}

	fmt.Println("Connected to Redis successfully! Response:", ping)

	// Set a key-value pair
	err = client.Set(ctx, "test_key", "Hello from CircleConnect Redis Test!", 0).Err()
	if err != nil {
		fmt.Printf("Error setting key: %v\n", err)
		return
	}
	fmt.Println("Successfully set test_key in Redis")

	// Retrieve the value
	val, err := client.Get(ctx, "test_key").Result()
	if err != nil {
		fmt.Printf("Error getting key: %v\n", err)
		return
	}

	fmt.Printf("Retrieved value for 'test_key': %s\n", val)
	fmt.Println("Redis test completed successfully!")
}
