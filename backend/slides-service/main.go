package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/martin226/slideitin/backend/slides-service/controllers"
	"github.com/martin226/slideitin/backend/slides-service/services/slides"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option" // Add option package
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Set up Gin router
	router := gin.Default()

	// Get environment variables
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}
	
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable is required")
	}

	// Initialize Firestore client
	ctx := context.Background()

	// Check for Firestore emulator host
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")

	var fsClient *firestore.Client
	// Note: 'err' is already declared by godotenv.Load() above, so no need to redeclare with :=

	if emulatorHost != "" {
		log.Printf("Using Firestore emulator at %s\n", emulatorHost)
		// Connect to the emulator
		fsClient, err = firestore.NewClient(ctx, projectID,
			option.WithEndpoint(emulatorHost),
			option.WithoutAuthentication(), // No credentials needed for emulator
		)
	} else {
		log.Println("Connecting to live Firestore")
		// Connect to live Firestore
		fsClient, err = firestore.NewClient(ctx, projectID)
	}

	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer fsClient.Close()
	
	// Initialize services
	slideService := slides.NewSlideService(apiKey)
	
	// Initialize controllers
	taskController := controllers.NewTaskController(slideService, fsClient)
	
	// Define routes
	router.POST("/tasks/process-slides", taskController.ProcessSlides)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Starting slides service on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
