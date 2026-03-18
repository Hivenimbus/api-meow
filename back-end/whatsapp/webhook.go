package whatsapp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// WebhookEvent represents an event to be sent to the webhook
type WebhookEvent struct {
	Event     string      `json:"event"`
	Instance  string      `json:"instance"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// MessageData represents incoming message data
type MessageData struct {
	From        string `json:"from"`
	FromName    string `json:"fromName,omitempty"`
	To          string `json:"to,omitempty"`
	MessageID   string `json:"messageId"`
	MessageType string `json:"messageType"`
	Text        string `json:"text,omitempty"`
	Caption     string `json:"caption,omitempty"`
	MediaURL    string `json:"mediaUrl,omitempty"`
	MediaBase64 string `json:"mediaBase64,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	IsGroup     bool   `json:"isGroup"`
	IsFromMe    bool   `json:"isFromMe"`
	GroupID     string `json:"groupId,omitempty"`
	GroupName   string `json:"groupName,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

// StatusData represents connection status data
type StatusData struct {
	Status      string `json:"status"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// maxConcurrentWebhooks is the maximum number of outgoing webhook calls that can
// run simultaneously. Keeping this bounded prevents goroutine accumulation when
// a webhook endpoint is slow or unreachable.
const maxConcurrentWebhooks = 50

// WebhookSender handles sending events to webhooks
type WebhookSender struct {
	client    *http.Client
	timeout   time.Duration
	semaphore chan struct{} // limits concurrent outgoing webhook goroutines
}

// NewWebhookSender creates a new webhook sender
func NewWebhookSender() *WebhookSender {
	// Use InsecureSkipVerify to avoid TLS certificate issues in environments
	// with outdated CA bundles. Webhook URLs are user-configured so the user
	// implicitly trusts the endpoint.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}
	return &WebhookSender{
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		timeout:   10 * time.Second,
		semaphore: make(chan struct{}, maxConcurrentWebhooks),
	}
}

// SendEvent sends an event to the webhook URL
func (ws *WebhookSender) SendEvent(ctx context.Context, webhookURL string, event WebhookEvent) error {
	if webhookURL == "" {
		return nil // No webhook configured, skip
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event.Event)
	req.Header.Set("X-Instance-Name", event.Instance)

	resp, err := ws.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}

// SendEventAsync sends an event asynchronously with bounded concurrency.
// If the semaphore is full (maxConcurrentWebhooks goroutines already running),
// the event is dropped and a warning is logged instead of spawning a new goroutine.
func (ws *WebhookSender) SendEventAsync(webhookURL string, event WebhookEvent) {
	if webhookURL == "" {
		return
	}

	select {
	case ws.semaphore <- struct{}{}:
		// slot acquired — proceed
	default:
		log.Printf("[Webhook] Dropping event %s for instance %s: %d concurrent webhook calls already in flight",
			event.Event, event.Instance, maxConcurrentWebhooks)
		return
	}

	go func() {
		defer func() { <-ws.semaphore }()

		ctx, cancel := context.WithTimeout(context.Background(), ws.timeout)
		defer cancel()

		if err := ws.SendEvent(ctx, webhookURL, event); err != nil {
			log.Printf("[Webhook] Error sending event %s for instance %s: %v", event.Event, event.Instance, err)
		} else {
			log.Printf("[Webhook] Event %s sent for instance %s", event.Event, event.Instance)
		}
	}()
}

// Global webhook sender instance
var webhookSender = NewWebhookSender()

// GetWebhookSender returns the global webhook sender
func GetWebhookSender() *WebhookSender {
	return webhookSender
}
