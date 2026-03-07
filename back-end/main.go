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
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	// Connect to database via GORM
	gormDB, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	// Create/update tables automatically
	if err := gormDB.AutoMigrate(&db.Instance{}, &db.Tag{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize WhatsApp manager
	waManager, err := whatsapp.NewInstanceManager(dbURL)
	if err != nil {
		log.Printf("Warning: Failed to initialize WhatsApp manager: %v", err)
		waManager = nil
	} else {
		defer waManager.Close()
		log.Println("WhatsApp manager initialized successfully")

		// Restore and auto-reconnect instances that were connected before restart
		go func() {
			var connectedInstances []db.Instance
			if err := gormDB.Where("status = ?", "connected").Find(&connectedInstances).Error; err != nil {
				log.Printf("Warning: Failed to get connected instances: %v", err)
				return
			}

			var instanceInfos []whatsapp.InstanceInfo
			for _, inst := range connectedInstances {
				if inst.PhoneNumber != nil && *inst.PhoneNumber != "" {
					info := whatsapp.InstanceInfo{
						Name:        inst.Name,
						PhoneNumber: *inst.PhoneNumber,
					}
					if inst.WebhookUrl != nil {
						info.WebhookURL = *inst.WebhookUrl
					}
					if inst.IgnoreGroups != nil {
						info.IgnoreGroups = *inst.IgnoreGroups
					}
					if inst.ProxyEnabled != nil && *inst.ProxyEnabled && inst.ProxyUrl != nil {
						info.ProxyURL = *inst.ProxyUrl
					}
					instanceInfos = append(instanceInfos, info)
				}
			}

			waManager.RestoreClients(instanceInfos)

			var wg sync.WaitGroup
			for _, inst := range instanceInfos {
				wg.Add(1)
				go func(instanceName string) {
					defer wg.Done()
					log.Printf("Auto-reconnecting instance: %s", instanceName)
					_, err := waManager.Connect(context.Background(), instanceName)
					if err != nil {
						log.Printf("Failed to auto-reconnect instance %s: %v", instanceName, err)
						gormDB.Model(&db.Instance{}).Where("name = ?", instanceName).Updates(map[string]interface{}{
							"status": "disconnected",
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

	app.Use(recover.New())

	app.Use(func(c *fiber.Ctx) error {
		log.Printf("%s %s", c.Method(), c.Path())
		return c.Next()
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API Meow - WhatsApp API Backend")
	})

	api := app.Group("/api", authMiddleware)

	// Instance routes
	api.Get("/instances", func(c *fiber.Ctx) error {
		var instances []db.Instance
		if err := gormDB.WithContext(c.Context()).Order("created_at desc").Find(&instances).Error; err != nil {
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

		instance := db.Instance{
			Name:            body.Name,
			WebhookUrl:      body.WebhookUrl,
			TagID:           body.TagId,
			IgnoreGroups:    &ignoreGroups,
			ReceiveMessages: &receiveMessages,
			ProxyEnabled:    &proxyEnabled,
			ProxyUrl:        body.ProxyUrl,
		}
		if err := gormDB.WithContext(c.Context()).Create(&instance).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(formatInstance(instance))
	})

	api.Get("/instances/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		var instance db.Instance
		if err := gormDB.WithContext(c.Context()).Where("name = ?", name).First(&instance).Error; err != nil {
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

		updates := map[string]interface{}{
			"status":       body.Status,
			"phone_number": body.PhoneNumber,
		}
		if err := gormDB.WithContext(c.Context()).Model(&db.Instance{}).Where("name = ?", name).Updates(updates).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		var instance db.Instance
		gormDB.WithContext(c.Context()).Where("name = ?", name).First(&instance)
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

		updates := map[string]interface{}{
			"webhook_url":      body.WebhookUrl,
			"ignore_groups":    body.IgnoreGroups,
			"receive_messages": body.ReceiveMessages,
			"tag_id":           body.TagId,
			"proxy_enabled":    body.ProxyEnabled,
			"proxy_url":        body.ProxyUrl,
		}
		if err := gormDB.WithContext(c.Context()).Model(&db.Instance{}).Where("name = ?", name).Updates(updates).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		var instance db.Instance
		gormDB.WithContext(c.Context()).Where("name = ?", name).First(&instance)
		return c.JSON(formatInstance(instance))
	})

	api.Delete("/instances/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")

		var instance db.Instance
		if err := gormDB.WithContext(c.Context()).Where("name = ?", name).First(&instance).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Instance not found"})
		}

		if waManager != nil {
			waManager.RemoveClient(name)
		}

		if err := gormDB.WithContext(c.Context()).Where("name = ?", name).Delete(&db.Instance{}).Error; err != nil {
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

		var instance db.Instance
		if err := gormDB.WithContext(c.Context()).Where("name = ?", name).First(&instance).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Instance not found"})
		}

		client, err := waManager.GetClient(name)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		if instance.WebhookUrl != nil {
			client.SetWebhook(*instance.WebhookUrl)
		}
		if instance.IgnoreGroups != nil {
			client.SetIgnoreGroups(*instance.IgnoreGroups)
		}
		if instance.ProxyEnabled != nil && *instance.ProxyEnabled && instance.ProxyUrl != nil {
			if err := client.SetProxy(*instance.ProxyUrl); err != nil {
				log.Printf("Failed to set proxy for instance %s: %v", name, err)
				return c.Status(400).JSON(fiber.Map{"error": "Invalid proxy URL"})
			}
		} else {
			_ = client.SetProxy("")
		}

		ctx := context.Background()
		err = client.Connect(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		gormDB.WithContext(c.Context()).Model(&db.Instance{}).Where("name = ?", name).Updates(map[string]interface{}{
			"status": "connecting",
		})

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

		if status == "connected" && phone != "" {
			gormDB.WithContext(c.Context()).Model(&db.Instance{}).Where("name = ?", name).Updates(map[string]interface{}{
				"status":       "connected",
				"phone_number": phone,
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

		gormDB.WithContext(c.Context()).Model(&db.Instance{}).Where("name = ?", name).Updates(map[string]interface{}{
			"status": "disconnected",
		})

		return c.JSON(fiber.Map{"message": "Disconnected successfully"})
	})

	// Set presence endpoint
	api.Post("/instances/:name/set-presence", func(c *fiber.Ctx) error {
		if waManager == nil {
			return c.Status(503).JSON(fiber.Map{"error": "WhatsApp service not available"})
		}

		name := c.Params("name")

		var body struct {
			Available bool `json:"available"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if err := waManager.SetPresence(name, body.Available); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "Presence updated"})
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
			TypingDuration int    `json:"typingDuration"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if body.To == "" || body.Text == "" {
			return c.Status(400).JSON(fiber.Map{"error": "to and text are required"})
		}

		resp, err := waManager.SendTextMessage(name, body.To, body.Text, body.SimulateTyping, body.TypingDuration)
		if err != nil {
			status := 500
			if strings.Contains(err.Error(), "WhatsApp") || strings.Contains(err.Error(), "recipient") || strings.Contains(err.Error(), "connected") {
				status = 400
			}
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"messageId": resp.MessageID,
			"timestamp": resp.Timestamp.Format(time.RFC3339),
		})
	})

	// Send text message batch endpoint
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
			MaxWorkers int `json:"maxWorkers"`
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
			MediaType         string `json:"mediaType"`
			Base64Data        string `json:"base64Data"`
			URL               string `json:"url"`
			MimeType          string `json:"mimeType"`
			Caption           string `json:"caption"`
			FileName          string `json:"fileName"`
			SimulateRecording bool   `json:"simulateRecording"`
			RecordingDuration int    `json:"recordingDuration"`
			PTT               bool   `json:"ptt"`
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
			mediaData, err = base64.StdEncoding.DecodeString(body.Base64Data)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid base64 data"})
			}
		}

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
			status := 500
			if strings.Contains(err.Error(), "WhatsApp") || strings.Contains(err.Error(), "recipient") || strings.Contains(err.Error(), "connected") {
				status = 400
			}
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"messageId": resp.MessageID,
			"timestamp": resp.Timestamp.Format(time.RFC3339),
		})
	})

	// Send media batch endpoint
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
			MaxWorkers int `json:"maxWorkers"`
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
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				result := BatchResult{Index: idx, To: message.To}

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

				mimeType := message.MimeType
				if mimeType == "" {
					mimeType = http.DetectContentType(mediaData)
				}

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
		var tags []db.Tag
		if err := gormDB.WithContext(c.Context()).Order("name").Find(&tags).Error; err != nil {
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

		tag := db.Tag{Name: body.Name, Color: body.Color}
		if err := gormDB.WithContext(c.Context()).Create(&tag).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(formatTag(tag))
	})

	api.Put("/tags/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		var body struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if err := gormDB.WithContext(c.Context()).Model(&db.Tag{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name":  body.Name,
			"color": body.Color,
		}).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		var tag db.Tag
		gormDB.WithContext(c.Context()).Where("id = ?", id).First(&tag)
		return c.JSON(formatTag(tag))
	})

	api.Delete("/tags/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		if err := gormDB.WithContext(c.Context()).Where("id = ?", id).Delete(&db.Tag{}).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	})

	log.Println("Server starting on port 8080...")
	log.Fatal(app.Listen(":8080"))
}

func authMiddleware(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Authorization header required"})
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if token != apiKey {
		return c.Status(403).JSON(fiber.Map{"error": "Invalid API key"})
	}

	return c.Next()
}

type InstanceResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	PhoneNumber     string  `json:"phoneNumber,omitempty"`
	TagId           string  `json:"tagId,omitempty"`
	IgnoreGroups    bool    `json:"ignoreGroups"`
	WebhookUrl      string  `json:"webhookUrl,omitempty"`
	ReceiveMessages bool    `json:"receiveMessages"`
	ProxyEnabled    bool    `json:"proxyEnabled"`
	ProxyUrl        string  `json:"proxyUrl,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

func formatInstance(i db.Instance) InstanceResponse {
	r := InstanceResponse{
		ID:        i.ID,
		Name:      i.Name,
		Status:    i.Status,
		CreatedAt: i.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: i.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if i.PhoneNumber != nil {
		r.PhoneNumber = *i.PhoneNumber
	}
	if i.TagID != nil {
		r.TagId = *i.TagID
	}
	if i.IgnoreGroups != nil {
		r.IgnoreGroups = *i.IgnoreGroups
	}
	if i.ReceiveMessages != nil {
		r.ReceiveMessages = *i.ReceiveMessages
	}
	if i.ProxyEnabled != nil {
		r.ProxyEnabled = *i.ProxyEnabled
	}
	if i.ProxyUrl != nil {
		r.ProxyUrl = *i.ProxyUrl
	}
	if i.WebhookUrl != nil {
		r.WebhookUrl = *i.WebhookUrl
	}
	return r
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
		ID:        t.ID,
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func formatTags(tags []db.Tag) []TagResponse {
	result := make([]TagResponse, len(tags))
	for i, tag := range tags {
		result[i] = formatTag(tag)
	}
	return result
}
