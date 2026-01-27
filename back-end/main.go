package main

import (
	"context"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"api-meow/internal/db"
	"api-meow/whatsapp"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
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

	// Initialize WhatsApp manager
	waManager, err := whatsapp.NewInstanceManager(dbURL)
	if err != nil {
		log.Printf("Warning: Failed to initialize WhatsApp manager: %v", err)
		// Continue without WhatsApp functionality
		waManager = nil
	} else {
		defer waManager.Close()
		log.Println("WhatsApp manager initialized successfully")

		// Restore and auto-reconnect instances that were connected before restart
		go func() {
			// Get all connected instances from database
			connectedInstances, err := queries.ListConnectedInstances(context.Background())
			if err != nil {
				log.Printf("Warning: Failed to get connected instances: %v", err)
				return
			}

			// Build instance info list for restoration
			var instanceInfos []whatsapp.InstanceInfo
			for _, inst := range connectedInstances {
				if inst.PhoneNumber.Valid && inst.PhoneNumber.String != "" {
					info := whatsapp.InstanceInfo{
						Name:        inst.Name,
						PhoneNumber: inst.PhoneNumber.String,
					}
					if inst.WebhookUrl.Valid {
						info.WebhookURL = inst.WebhookUrl.String
					}
					if inst.IgnoreGroups.Valid {
						info.IgnoreGroups = inst.IgnoreGroups.Bool
					}
					if inst.ProxyEnabled.Bool && inst.ProxyUrl.Valid {
						info.ProxyURL = inst.ProxyUrl.String
					}
					instanceInfos = append(instanceInfos, info)
				}
			}

			// Restore clients using correct name mapping
			waManager.RestoreClients(instanceInfos)

			// Auto-reconnect restored instances IN PARALLEL
			var wg sync.WaitGroup
			for _, inst := range instanceInfos {
				wg.Add(1)
				go func(instanceName string) {
					defer wg.Done()
					log.Printf("Auto-reconnecting instance: %s", instanceName)
					_, err := waManager.Connect(context.Background(), instanceName)
					if err != nil {
						log.Printf("Failed to auto-reconnect instance %s: %v", instanceName, err)
						// Update status in database
						queries.UpdateInstanceStatusByName(context.Background(), db.UpdateInstanceStatusByNameParams{
							Name:   instanceName,
							Status: "disconnected",
						})
					} else {
						log.Printf("Successfully auto-reconnected instance: %s", instanceName)
					}
				}(inst.Name)
			}
			wg.Wait()
			log.Printf("All instances reconnection completed")
		}()
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Printf("Error: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// Recover Middleware
	app.Use(recover.New())

	// Request Logging (Simple)
	app.Use(func(c *fiber.Ctx) error {
		log.Printf("%s %s", c.Method(), c.Path())
		return c.Next()
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
			ProxyEnabled    *bool   `json:"proxyEnabled"`
			ProxyUrl        *string `json:"proxyUrl"`
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
		proxyEnabled := false
		if body.ProxyEnabled != nil {
			proxyEnabled = *body.ProxyEnabled
		}

		instance, err := queries.CreateInstance(c.Context(), db.CreateInstanceParams{
			Name:            body.Name,
			WebhookUrl:      textFromPtr(body.WebhookUrl),
			TagID:           textFromPtr(body.TagId),
			IgnoreGroups:    pgtype.Bool{Bool: ignoreGroups, Valid: true},
			ReceiveMessages: pgtype.Bool{Bool: receiveMessages, Valid: true},
			ProxyEnabled:    pgtype.Bool{Bool: proxyEnabled, Valid: true},
			ProxyUrl:        textFromPtr(body.ProxyUrl),
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(formatInstance(instance))
	})

	api.Get("/instances/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		instance, err := queries.GetInstanceByName(c.Context(), name)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Instance not found"})
		}
		return c.JSON(formatInstance(instance))
	})

	api.Put("/instances/:name/status", func(c *fiber.Ctx) error {
		name := c.Params("name")

		var body struct {
			Status      string  `json:"status"`
			PhoneNumber *string `json:"phoneNumber"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		instance, err := queries.UpdateInstanceStatusByName(c.Context(), db.UpdateInstanceStatusByNameParams{
			Name:        name,
			Status:      body.Status,
			PhoneNumber: textFromPtr(body.PhoneNumber),
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(formatInstance(instance))
	})

	api.Put("/instances/:name/settings", func(c *fiber.Ctx) error {
		name := c.Params("name")

		var body struct {
			WebhookUrl      *string `json:"webhookUrl"`
			IgnoreGroups    bool    `json:"ignoreGroups"`
			ReceiveMessages bool    `json:"receiveMessages"`
			TagId           *string `json:"tagId"`
			ProxyEnabled    bool    `json:"proxyEnabled"`
			ProxyUrl        *string `json:"proxyUrl"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		instance, err := queries.UpdateInstanceSettingsByName(c.Context(), db.UpdateInstanceSettingsByNameParams{
			Name:            name,
			WebhookUrl:      textFromPtr(body.WebhookUrl),
			IgnoreGroups:    pgtype.Bool{Bool: body.IgnoreGroups, Valid: true},
			ReceiveMessages: pgtype.Bool{Bool: body.ReceiveMessages, Valid: true},
			TagID:           textFromPtr(body.TagId),
			ProxyEnabled:    pgtype.Bool{Bool: body.ProxyEnabled, Valid: true},
			ProxyUrl:        textFromPtr(body.ProxyUrl),
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(formatInstance(instance))
	})

	api.Delete("/instances/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")

		// Disconnect and remove WhatsApp client if exists
		if waManager != nil {
			waManager.RemoveClient(name)
		}

		err := queries.DeleteInstanceByName(c.Context(), name)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	})

	// WhatsApp connection routes
	api.Post("/instances/:name/connect", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		// Verify instance exists
		instance, err := queries.GetInstanceByName(c.Context(), name)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Instance not found"})
		}

		// Get client instance (creates it if needed)
		client, err := waManager.GetClient(name)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Configure settings BEFORE connecting
		if instance.WebhookUrl.Valid {
			client.SetWebhook(instance.WebhookUrl.String)
		}
		if instance.IgnoreGroups.Valid {
			client.SetIgnoreGroups(instance.IgnoreGroups.Bool)
		}
		if instance.ProxyEnabled.Bool && instance.ProxyUrl.Valid {
			if err := client.SetProxy(instance.ProxyUrl.String); err != nil {
				log.Printf("Failed to set proxy for instance %s: %v", name, err)
				return c.Status(400).JSON(fiber.Map{"error": "Invalid proxy URL"})
			}
		} else {
			// Ensure proxy is cleared if disabled/empty
			_ = client.SetProxy("")
		}

		// Start connection (will generate QR code if needed)
		// We use a background context for the connection to ensure it persists
		ctx := context.Background()
		err = client.Connect(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Update instance status to connecting
		queries.UpdateInstanceStatusByName(c.Context(), db.UpdateInstanceStatusByNameParams{
			Name:   name,
			Status: "connecting",
		})

		// Wait a bit for QR code to be generated
		time.Sleep(500 * time.Millisecond)

		qrCode := client.GetQRCode()
		status, phone := client.GetStatus()

		return c.JSON(fiber.Map{
			"status":  status,
			"qrCode":  qrCode,
			"phone":   phone,
			"message": "Connection initiated",
		})
	})

	api.Get("/instances/:name/qrcode", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		qrCode, err := waManager.GetQRCode(name)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}

		status, phone, _ := waManager.GetStatus(name)

		return c.JSON(fiber.Map{
			"qrCode": qrCode,
			"status": status,
			"phone":  phone,
		})
	})

	api.Get("/instances/:name/wa-status", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		status, phone, _ := waManager.GetStatus(name)

		// If connected and phone changed, update database
		if status == "connected" && phone != "" {
			queries.UpdateInstanceStatusByName(c.Context(), db.UpdateInstanceStatusByNameParams{
				Name:        name,
				Status:      "connected",
				PhoneNumber: pgtype.Text{String: phone, Valid: true},
			})
		}

		return c.JSON(fiber.Map{
			"status": status,
			"phone":  phone,
		})
	})

	api.Post("/instances/:name/disconnect", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		err := waManager.Disconnect(name)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Update instance status
		queries.UpdateInstanceStatusByName(c.Context(), db.UpdateInstanceStatusByNameParams{
			Name:   name,
			Status: "disconnected",
		})

		return c.JSON(fiber.Map{"message": "Disconnected successfully"})
	})

	// Send text message endpoint
	api.Post("/instances/:name/send-message", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		var body struct {
			To             string `json:"to"`
			Text           string `json:"text"`
			SimulateTyping bool   `json:"simulateTyping"`
			TypingDuration int    `json:"typingDuration"` // milliseconds
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if body.To == "" || body.Text == "" {
			return c.Status(400).JSON(fiber.Map{"error": "to and text are required"})
		}

		resp, err := waManager.SendTextMessage(name, body.To, body.Text, body.SimulateTyping, body.TypingDuration)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"messageId": resp.MessageID,
			"timestamp": resp.Timestamp.Format(time.RFC3339),
		})
	})

	// Send text message batch endpoint - uses worker pool for parallel processing
	api.Post("/instances/:name/send-message-batch", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		var body struct {
			Messages []struct {
				To             string `json:"to"`
				Text           string `json:"text"`
				SimulateTyping bool   `json:"simulateTyping"`
				TypingDuration int    `json:"typingDuration"`
			} `json:"messages"`
			MaxWorkers int `json:"maxWorkers"` // Number of parallel workers (default: 5)
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if len(body.Messages) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "messages array is required"})
		}

		maxWorkers := body.MaxWorkers
		if maxWorkers <= 0 {
			maxWorkers = 5
		}
		if maxWorkers > 20 {
			maxWorkers = 20
		}

		type BatchResult struct {
			Index     int    `json:"index"`
			To        string `json:"to"`
			MessageID string `json:"messageId,omitempty"`
			Error     string `json:"error,omitempty"`
			Success   bool   `json:"success"`
		}

		results := make([]BatchResult, len(body.Messages))
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, maxWorkers)

		for i, msg := range body.Messages {
			wg.Add(1)
			go func(idx int, message struct {
				To             string `json:"to"`
				Text           string `json:"text"`
				SimulateTyping bool   `json:"simulateTyping"`
				TypingDuration int    `json:"typingDuration"`
			}) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				result := BatchResult{Index: idx, To: message.To}

				if message.To == "" || message.Text == "" {
					result.Error = "to and text are required"
					results[idx] = result
					return
				}

				resp, err := waManager.SendTextMessage(name, message.To, message.Text, message.SimulateTyping, message.TypingDuration)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Success = true
					result.MessageID = resp.MessageID
				}
				results[idx] = result
			}(i, msg)
		}

		wg.Wait()

		successCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			}
		}

		return c.JSON(fiber.Map{
			"total":   len(body.Messages),
			"success": successCount,
			"failed":  len(body.Messages) - successCount,
			"results": results,
		})
	})

	// Send media endpoint
	api.Post("/instances/:name/send-media", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		var body struct {
			To                string `json:"to"`
			MediaType         string `json:"mediaType"` // image, video, audio, document
			Base64Data        string `json:"base64Data"`
			URL               string `json:"url"` // Alternative to base64Data
			MimeType          string `json:"mimeType"`
			Caption           string `json:"caption"`           // For image, video, document
			FileName          string `json:"fileName"`          // For document
			SimulateRecording bool   `json:"simulateRecording"` // For audio
			RecordingDuration int    `json:"recordingDuration"` // For audio (ms)
			PTT               bool   `json:"ptt"`               // For audio (push-to-talk)
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if body.To == "" || body.MediaType == "" {
			return c.Status(400).JSON(fiber.Map{"error": "to and mediaType are required"})
		}

		if body.Base64Data == "" && body.URL == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Either base64Data or url must be provided"})
		}

		var mediaData []byte
		var err error

		if body.URL != "" {
			// Download from URL
			httpResp, err := http.Get(body.URL)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Failed to download from URL: " + err.Error()})
			}
			defer httpResp.Body.Close()

			if httpResp.StatusCode != http.StatusOK {
				return c.Status(400).JSON(fiber.Map{"error": "Failed to download from URL: HTTP " + httpResp.Status})
			}

			mediaData, err = io.ReadAll(httpResp.Body)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Failed to read URL response: " + err.Error()})
			}
		} else {
			// Decode base64 data
			mediaData, err = base64.StdEncoding.DecodeString(body.Base64Data)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid base64 data"})
			}
		}

		// Auto-detect mimeType if not provided
		mimeType := body.MimeType
		if mimeType == "" {
			mimeType = http.DetectContentType(mediaData)
		}

		var resp *whatsapp.SendResponse

		switch body.MediaType {
		case "image":
			resp, err = waManager.SendImageMessage(name, body.To, mediaData, mimeType, body.Caption)
		case "video":
			resp, err = waManager.SendVideoMessage(name, body.To, mediaData, mimeType, body.Caption)
		case "audio":
			resp, err = waManager.SendAudioMessage(name, body.To, mediaData, mimeType, body.SimulateRecording, body.RecordingDuration, body.PTT)
		case "document":
			resp, err = waManager.SendDocumentMessage(name, body.To, mediaData, mimeType, body.FileName, body.Caption)
		default:
			return c.Status(400).JSON(fiber.Map{"error": "Invalid mediaType. Must be: image, video, audio, or document"})
		}

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"messageId": resp.MessageID,
			"timestamp": resp.Timestamp.Format(time.RFC3339),
		})
	})

	// Send media batch endpoint - uses worker pool for parallel processing
	api.Post("/instances/:name/send-media-batch", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		var body struct {
			Messages []struct {
				To                string `json:"to"`
				MediaType         string `json:"mediaType"`
				Base64Data        string `json:"base64Data"`
				URL               string `json:"url"`
				MimeType          string `json:"mimeType"`
				Caption           string `json:"caption"`
				FileName          string `json:"fileName"`
				SimulateRecording bool   `json:"simulateRecording"`
				RecordingDuration int    `json:"recordingDuration"`
				PTT               bool   `json:"ptt"`
			} `json:"messages"`
			MaxWorkers int `json:"maxWorkers"` // Number of parallel workers (default: 5)
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if len(body.Messages) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "messages array is required"})
		}

		// Set default max workers
		maxWorkers := body.MaxWorkers
		if maxWorkers <= 0 {
			maxWorkers = 5
		}
		if maxWorkers > 20 {
			maxWorkers = 20 // Cap at 20 workers
		}

		// Results channel
		type BatchResult struct {
			Index     int    `json:"index"`
			To        string `json:"to"`
			MessageID string `json:"messageId,omitempty"`
			Error     string `json:"error,omitempty"`
			Success   bool   `json:"success"`
		}

		results := make([]BatchResult, len(body.Messages))
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, maxWorkers)

		for i, msg := range body.Messages {
			wg.Add(1)
			go func(idx int, message struct {
				To                string `json:"to"`
				MediaType         string `json:"mediaType"`
				Base64Data        string `json:"base64Data"`
				URL               string `json:"url"`
				MimeType          string `json:"mimeType"`
				Caption           string `json:"caption"`
				FileName          string `json:"fileName"`
				SimulateRecording bool   `json:"simulateRecording"`
				RecordingDuration int    `json:"recordingDuration"`
				PTT               bool   `json:"ptt"`
			}) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				result := BatchResult{Index: idx, To: message.To}

				// Validate
				if message.To == "" || message.MediaType == "" {
					result.Error = "to and mediaType are required"
					results[idx] = result
					return
				}

				if message.Base64Data == "" && message.URL == "" {
					result.Error = "Either base64Data or url must be provided"
					results[idx] = result
					return
				}

				// Get media data
				var mediaData []byte
				var err error

				if message.URL != "" {
					httpResp, err := http.Get(message.URL)
					if err != nil {
						result.Error = "Failed to download: " + err.Error()
						results[idx] = result
						return
					}
					defer httpResp.Body.Close()

					if httpResp.StatusCode != http.StatusOK {
						result.Error = "Download failed: HTTP " + httpResp.Status
						results[idx] = result
						return
					}

					mediaData, err = io.ReadAll(httpResp.Body)
					if err != nil {
						result.Error = "Failed to read response: " + err.Error()
						results[idx] = result
						return
					}
				} else {
					mediaData, err = base64.StdEncoding.DecodeString(message.Base64Data)
					if err != nil {
						result.Error = "Invalid base64 data"
						results[idx] = result
						return
					}
				}

				// Auto-detect mimeType
				mimeType := message.MimeType
				if mimeType == "" {
					mimeType = http.DetectContentType(mediaData)
				}

				// Send message
				var resp *whatsapp.SendResponse
				switch message.MediaType {
				case "image":
					resp, err = waManager.SendImageMessage(name, message.To, mediaData, mimeType, message.Caption)
				case "video":
					resp, err = waManager.SendVideoMessage(name, message.To, mediaData, mimeType, message.Caption)
				case "audio":
					resp, err = waManager.SendAudioMessage(name, message.To, mediaData, mimeType, message.SimulateRecording, message.RecordingDuration, message.PTT)
				case "document":
					resp, err = waManager.SendDocumentMessage(name, message.To, mediaData, mimeType, message.FileName, message.Caption)
				default:
					result.Error = "Invalid mediaType"
					results[idx] = result
					return
				}

				if err != nil {
					result.Error = err.Error()
				} else {
					result.Success = true
					result.MessageID = resp.MessageID
				}
				results[idx] = result
			}(i, msg)
		}

		wg.Wait()

		// Count successes
		successCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			}
		}

		return c.JSON(fiber.Map{
			"total":   len(body.Messages),
			"success": successCount,
			"failed":  len(body.Messages) - successCount,
			"results": results,
		})
	})

	// Get contacts endpoint
	api.Get("/instances/:name/contacts", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		contacts, err := waManager.GetContacts(name)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"contacts": contacts,
		})
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

	log.Println("Server starting on port 80...")
	log.Fatal(app.Listen(":80"))
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
	ProxyEnabled    bool   `json:"proxyEnabled"`
	ProxyUrl        string `json:"proxyUrl,omitempty"`
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
		ProxyEnabled:    i.ProxyEnabled.Bool,
		ProxyUrl:        i.ProxyUrl.String,
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
