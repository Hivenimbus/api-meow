package main

import (
	"context"
	"log"
	"os"

	"api-meow/internal/db"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env from parent directory
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: No .env file found or error loading it")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Connect to database
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Initialize queries
	queries := db.New(pool)
	_ = queries // Prevent unused error for now

	app := fiber.New()

	// Middleware
	app.Use(cors.New())

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, API Meow with Fiber & PGX!")
	})

	app.Get("/api/instances", func(c *fiber.Ctx) error {
		instances, err := queries.ListInstances(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(instances)
	})

	log.Println("Server starting on port 8080...")
	log.Fatal(app.Listen(":8080"))
}
