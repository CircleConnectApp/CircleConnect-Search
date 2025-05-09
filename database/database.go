package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	PgDB        *gorm.DB
	MongoDB     *mongo.Database
	MongoClient *mongo.Client
)

// InitDBs initializes both PostgreSQL and MongoDB connections
func InitDBs() {
	initPostgres()
	initMongoDB()

	// Initialize MongoDB indexes for search functionality
	InitIndexes()
}

// initPostgres initializes PostgreSQL connection
func initPostgres() {
	// Get connection parameters from environment variables or use defaults
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "yehia")
	dbName := getEnv("POSTGRES_DB", "circleConnect")
	dbPort := getEnv("POSTGRES_PORT", "5432")

	connectionString := "host=" + dbHost +
		" user=" + dbUser +
		" password=" + dbPassword +
		" dbname=" + dbName +
		" port=" + dbPort +
		" sslmode=disable"

	var err error
	PgDB, err = gorm.Open(postgres.Open(connectionString), &gorm.Config{})

	if err != nil {
		log.Fatal("Could not connect to the PostgreSQL database: ", err)
	}

	log.Println("Successfully connected to the PostgreSQL database")
}

// initMongoDB initializes MongoDB connection
func initMongoDB() {
	// Get connection parameters from environment variables or use defaults
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	mongoDBName := getEnv("MONGO_DB", "circleconnect_search")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a new client and connect to the server
	var err error
	MongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Could not connect to the MongoDB database: ", err)
	}

	// Check the connection
	err = MongoClient.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Could not ping the MongoDB database: ", err)
	}

	MongoDB = MongoClient.Database(mongoDBName)
	log.Println("Successfully connected to the MongoDB database")
}

// CloseMongoConnection closes the MongoDB connection when the service shuts down
func CloseMongoConnection() {
	if MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := MongoClient.Disconnect(ctx); err != nil {
			log.Fatal("Error closing MongoDB connection: ", err)
		}
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
