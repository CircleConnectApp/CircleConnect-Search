#!/bin/bash

# CircleConnect Search Service Deployment Script

set -e

# Display banner
echo "================================================="
echo "   CircleConnect Search Service Deployment"
echo "================================================="

# Check environment argument
if [ "$1" == "prod" ] || [ "$1" == "production" ]; then
    ENV="production"
    COMPOSE_FILE="docker-compose.prod.yml"
    ENV_FILE="prod.env"
    
    # Check if prod.env exists
    if [ ! -f "$ENV_FILE" ]; then
        echo "Error: $ENV_FILE not found. Please create it from prod.env.example"
        exit 1
    fi
    
    echo "Deploying in PRODUCTION mode"
else
    ENV="development"
    COMPOSE_FILE="docker-compose.yml"
    echo "Deploying in DEVELOPMENT mode"
fi

# Build and start containers
if [ "$ENV" == "production" ]; then
    echo "Building and starting containers using $COMPOSE_FILE with $ENV_FILE..."
    docker-compose -f $COMPOSE_FILE --env-file $ENV_FILE up -d --build
else
    echo "Building and starting containers using $COMPOSE_FILE..."
    docker-compose -f $COMPOSE_FILE up -d --build
fi

# Check if containers are running
echo "Checking services..."
sleep 5

if [ "$ENV" == "production" ]; then
    # Check specific services
    services=("user-search-service" "search-redis" "search-postgres" "search-mongodb")
    
    for service in "${services[@]}"; do
        if [ "$(docker ps -q -f name=$service)" ]; then
            echo "✅ $service is running"
        else
            echo "❌ $service is not running"
            echo "Deployment failed. Check logs with: docker-compose -f $COMPOSE_FILE logs"
            exit 1
        fi
    done
else
    # Check all services defined in docker-compose.yml
    if docker-compose -f $COMPOSE_FILE ps | grep -q "Exit"; then
        echo "❌ Some services failed to start"
        echo "Deployment failed. Check logs with: docker-compose -f $COMPOSE_FILE logs"
        exit 1
    else
        echo "✅ All services are running"
    fi
fi

echo ""
echo "================================================="
echo "   Deployment completed successfully!"
echo "================================================="
echo ""

if [ "$ENV" == "development" ]; then
    echo "Development API accessible at: http://localhost:8084"
else
    echo "Production API accessible at: http://localhost:8084"
    echo "Note: Configure your production reverse proxy as needed"
fi

echo ""
echo "Useful commands:"
echo "- View logs: docker-compose -f $COMPOSE_FILE logs -f"
echo "- Stop services: docker-compose -f $COMPOSE_FILE down"
echo "- Restart services: docker-compose -f $COMPOSE_FILE restart"
echo "" 