package main

import (
	"context"
	"log"
	"os"
	"strings"

	"api-meow/internal/db"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var apiKey string

func main() {
	// Load .env from parent directory
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: No .env file found or error loading it")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	apiKey = os.Getenv("APIKEY")
	if apiKey == "" {
		log.Fatal("APIKEY is not set")
	}

	// Connect to database
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Initialize queries
	queries := db.New(pool)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// CORS Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Public route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API Meow - WhatsApp API Backend")
	})

	// API routes with auth middleware
	api := app.Group("/api", authMiddleware)

	// Instance routes
	api.Get("/instances", func(c *fiber.Ctx) error {
		instances, err := queries.ListInstances(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(formatInstances(instances))
	})

	api.Post("/instances", func(c *fiber.Ctx) error {
		var body struct {
			Name            string  `json:"name"`
			WebhookUrl      *string `json:"webhookUrl"`
			TagId           *string `json:"tagId"`
			IgnoreGroups    *bool   `json:"ignoreGroups"`
			ReceiveMessages *bool   `json:"receiveMessages"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if body.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Name is required"})
		}

		ignoreGroups := true
		if body.IgnoreGroups != nil {
			ignoreGroups = *body.IgnoreGroups
		}
		receiveMessages := true
		if body.ReceiveMessages != nil {
			receiveMessages = *body.ReceiveMessages
		}

		instance, err := queries.CreateInstance(c.Context(), db.CreateInstanceParams{
			Name:            body.Name,
			WebhookUrl:      textFromPtr(body.WebhookUrl),
			TagID:           textFromPtr(body.TagId),
			IgnoreGroups:    pgtype.Bool{Bool: ignoreGroups, Valid: true},
			ReceiveMessages: pgtype.Bool{Bool: receiveMessages, Valid: true},
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(formatInstance(instance))
	})

	api.Get("/instances/:id", func(c *fiber.Ctx) error {
		id, err := parseUUID(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
		}

		instance, err := queries.GetInstance(c.Context(), id)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Instance not found"})
		}
		return c.JSON(formatInstance(instance))
	})

	api.Put("/instances/:id/status", func(c *fiber.Ctx) error {
		id, err := parseUUID(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
		}

		var body struct {
			Status      string  `json:"status"`
			PhoneNumber *string `json:"phoneNumber"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		instance, err := queries.UpdateInstanceStatus(c.Context(), db.UpdateInstanceStatusParams{
			ID:          id,
			Status:      body.Status,
			PhoneNumber: textFromPtr(body.PhoneNumber),
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(formatInstance(instance))
	})

	api.Put("/instances/:id/settings", func(c *fiber.Ctx) error {
		id, err := parseUUID(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
		}

		var body struct {
			WebhookUrl      *string `json:"webhookUrl"`
			IgnoreGroups    bool    `json:"ignoreGroups"`
			ReceiveMessages bool    `json:"receiveMessages"`
			TagId           *string `json:"tagId"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		instance, err := queries.UpdateInstanceSettings(c.Context(), db.UpdateInstanceSettingsParams{
			ID:              id,
			WebhookUrl:      textFromPtr(body.WebhookUrl),
			IgnoreGroups:    pgtype.Bool{Bool: body.IgnoreGroups, Valid: true},
			ReceiveMessages: pgtype.Bool{Bool: body.ReceiveMessages, Valid: true},
			TagID:           textFromPtr(body.TagId),
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(formatInstance(instance))
	})

	api.Delete("/instances/:id", func(c *fiber.Ctx) error {
		id, err := parseUUID(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
		}

		err = queries.DeleteInstance(c.Context(), id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	})

	// Tag routes
	api.Get("/tags", func(c *fiber.Ctx) error {
		tags, err := queries.ListTags(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(formatTags(tags))
	})

	api.Post("/tags", func(c *fiber.Ctx) error {
		var body struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if body.Name == "" || body.Color == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Name and color are required"})
		}

		tag, err := queries.CreateTag(c.Context(), db.CreateTagParams{
			Name:  body.Name,
			Color: body.Color,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(formatTag(tag))
	})

	api.Put("/tags/:id", func(c *fiber.Ctx) error {
		id, err := parseUUID(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
		}

		var body struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		tag, err := queries.UpdateTag(c.Context(), db.UpdateTagParams{
			ID:    id,
			Name:  body.Name,
			Color: body.Color,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(formatTag(tag))
	})

	api.Delete("/tags/:id", func(c *fiber.Ctx) error {
		id, err := parseUUID(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
		}

		err = queries.DeleteTag(c.Context(), id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	})

	log.Println("Server starting on port 8080...")
	log.Fatal(app.Listen(":8080"))
}

// Auth middleware
func authMiddleware(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Authorization header required"})
	}

	// Support both "Bearer <token>" and just "<token>"
	token := strings.TrimPrefix(auth, "Bearer ")
	if token != apiKey {
		return c.Status(403).JSON(fiber.Map{"error": "Invalid API key"})
	}

	return c.Next()
}

// Helper functions
func parseUUID(s string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	err := uuid.Scan(s)
	return uuid, err
}

func textFromPtr(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return strings.ToLower(strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(
						string([]byte{
							hexChar(b[0] >> 4), hexChar(b[0] & 0xf),
							hexChar(b[1] >> 4), hexChar(b[1] & 0xf),
							hexChar(b[2] >> 4), hexChar(b[2] & 0xf),
							hexChar(b[3] >> 4), hexChar(b[3] & 0xf),
							'-',
							hexChar(b[4] >> 4), hexChar(b[4] & 0xf),
							hexChar(b[5] >> 4), hexChar(b[5] & 0xf),
							'-',
							hexChar(b[6] >> 4), hexChar(b[6] & 0xf),
							hexChar(b[7] >> 4), hexChar(b[7] & 0xf),
							'-',
							hexChar(b[8] >> 4), hexChar(b[8] & 0xf),
							hexChar(b[9] >> 4), hexChar(b[9] & 0xf),
							'-',
							hexChar(b[10] >> 4), hexChar(b[10] & 0xf),
							hexChar(b[11] >> 4), hexChar(b[11] & 0xf),
							hexChar(b[12] >> 4), hexChar(b[12] & 0xf),
							hexChar(b[13] >> 4), hexChar(b[13] & 0xf),
							hexChar(b[14] >> 4), hexChar(b[14] & 0xf),
							hexChar(b[15] >> 4), hexChar(b[15] & 0xf),
						}),
						"\x00", ""),
					"\x00", ""),
				"\x00", ""),
			"\x00", ""),
		"\x00", ""))
}

func hexChar(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + b - 10
}

type InstanceResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	PhoneNumber     string `json:"phoneNumber,omitempty"`
	TagId           string `json:"tagId,omitempty"`
	IgnoreGroups    bool   `json:"ignoreGroups"`
	WebhookUrl      string `json:"webhookUrl,omitempty"`
	ReceiveMessages bool   `json:"receiveMessages"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func formatInstance(i db.Instance) InstanceResponse {
	return InstanceResponse{
		ID:              uuidToString(i.ID),
		Name:            i.Name,
		Status:          i.Status,
		PhoneNumber:     i.PhoneNumber.String,
		TagId:           i.TagID.String,
		IgnoreGroups:    i.IgnoreGroups.Bool,
		WebhookUrl:      i.WebhookUrl.String,
		ReceiveMessages: i.ReceiveMessages.Bool,
		CreatedAt:       i.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       i.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func formatInstances(instances []db.Instance) []InstanceResponse {
	result := make([]InstanceResponse, len(instances))
	for i, inst := range instances {
		result[i] = formatInstance(inst)
	}
	return result
}

type TagResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedAt string `json:"createdAt"`
}

func formatTag(t db.Tag) TagResponse {
	return TagResponse{
		ID:        uuidToString(t.ID),
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: t.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func formatTags(tags []db.Tag) []TagResponse {
	result := make([]TagResponse, len(tags))
	for i, tag := range tags {
		result[i] = formatTag(tag)
	}
	return result
}
