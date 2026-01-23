package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// WAClient wraps a whatsmeow client with additional functionality
type WAClient struct {
	instanceID string
	client     *whatsmeow.Client
	qrCode     string
	status     string // disconnected, connecting, connected
	phone      string
	mu         sync.RWMutex

	// Channels for events
	QRCodeChan    chan string
	ConnectedChan chan bool
	ErrorChan     chan error
}

// NewWAClient creates a new WhatsApp client wrapper
func NewWAClient(instanceID string, client *whatsmeow.Client) *WAClient {
	wac := &WAClient{
		instanceID:    instanceID,
		client:        client,
		status:        "disconnected",
		QRCodeChan:    make(chan string, 10),
		ConnectedChan: make(chan bool, 1),
		ErrorChan:     make(chan error, 10),
	}

	// Register event handler
	client.AddEventHandler(wac.eventHandler)

	return wac
}

// Connect initiates the connection to WhatsApp
func (w *WAClient) Connect(ctx context.Context) error {
	w.mu.Lock()
	w.status = "connecting"
	w.mu.Unlock()

	// Check if already logged in
	if w.client.Store.ID != nil {
		// Already registered, just connect
		err := w.client.Connect()
		if err != nil {
			w.mu.Lock()
			w.status = "disconnected"
			w.mu.Unlock()
			return err
		}
		return nil
	}

	// Not logged in, need QR code
	qrChan, _ := w.client.GetQRChannel(ctx)
	err := w.client.Connect()
	if err != nil {
		w.mu.Lock()
		w.status = "disconnected"
		w.mu.Unlock()
		return err
	}

	// Handle QR codes in background
	go w.handleQRCodes(ctx, qrChan)

	return nil
}

// handleQRCodes processes QR codes from the channel
func (w *WAClient) handleQRCodes(ctx context.Context, qrChan <-chan whatsmeow.QRChannelItem) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-qrChan:
			if !ok {
				return
			}

			if evt.Event == "code" {
				// Generate QR code image as base64
				qrPng, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
				if err != nil {
					w.ErrorChan <- fmt.Errorf("failed to generate QR code: %w", err)
					continue
				}

				qrBase64 := base64.StdEncoding.EncodeToString(qrPng)

				w.mu.Lock()
				w.qrCode = qrBase64
				w.status = "connecting"
				w.mu.Unlock()

				// Notify through channel
				select {
				case w.QRCodeChan <- qrBase64:
				default:
					// Channel full, skip
				}
			} else if evt.Event == "success" {
				w.mu.Lock()
				w.qrCode = ""
				w.status = "connected"
				w.mu.Unlock()

				select {
				case w.ConnectedChan <- true:
				default:
				}
				return
			} else if evt.Event == "timeout" {
				w.mu.Lock()
				w.qrCode = ""
				w.status = "disconnected"
				w.mu.Unlock()
				return
			}
		}
	}
}

// eventHandler handles WhatsApp events
func (w *WAClient) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		w.mu.Lock()
		w.status = "connected"
		if w.client.Store.ID != nil {
			w.phone = w.client.Store.ID.User
		}
		w.mu.Unlock()

		select {
		case w.ConnectedChan <- true:
		default:
		}

	case *events.Disconnected:
		w.mu.Lock()
		w.status = "disconnected"
		w.qrCode = ""
		w.mu.Unlock()

	case *events.LoggedOut:
		w.mu.Lock()
		w.status = "disconnected"
		w.phone = ""
		w.qrCode = ""
		w.mu.Unlock()

	case *events.PairSuccess:
		w.mu.Lock()
		w.status = "connected"
		w.phone = v.ID.User
		w.qrCode = ""
		w.mu.Unlock()

	case *events.StreamError:
		w.ErrorChan <- fmt.Errorf("stream error: %s", v.Code)
	}
}

// Disconnect disconnects from WhatsApp
func (w *WAClient) Disconnect() {
	w.client.Disconnect()
	w.mu.Lock()
	w.status = "disconnected"
	w.mu.Unlock()
}

// Logout logs out and removes the session
func (w *WAClient) Logout(ctx context.Context) error {
	err := w.client.Logout(ctx)
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.status = "disconnected"
	w.phone = ""
	w.qrCode = ""
	w.mu.Unlock()
	return nil
}

// GetQRCode returns the current QR code as base64
func (w *WAClient) GetQRCode() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.qrCode
}

// GetStatus returns the current connection status and phone number
func (w *WAClient) GetStatus() (string, string) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status, w.phone
}

// IsConnected returns whether the client is connected
func (w *WAClient) IsConnected() bool {
	return w.client.IsConnected()
}

// WaitForConnection waits for connection with timeout
func (w *WAClient) WaitForConnection(timeout time.Duration) error {
	select {
	case <-w.ConnectedChan:
		return nil
	case err := <-w.ErrorChan:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("connection timeout")
	}
}

// GetPhoneNumber returns the phone number if connected
func (w *WAClient) GetPhoneNumber() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.phone
}
