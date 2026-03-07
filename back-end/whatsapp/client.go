package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"strings"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// WAClient wraps a whatsmeow client with additional functionality
type WAClient struct {
	instanceID   string
	client       *whatsmeow.Client
	qrCode       string
	status       string // disconnected, connecting, connected
	phone        string
	webhookURL   string
	ignoreGroups bool
	connectedAt  time.Time // Track when client connected to filter offline messages
	mu           sync.RWMutex

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

	// If status is "connecting" (e.g., QR code displayed but not scanned),
	// disconnect first to allow a fresh reconnection.
	// NOTE: IsConnected() returns true even during QR scan (WebSocket is open),
	// so we cannot rely on it to distinguish "waiting for QR" from "actually connecting".
	// The safest approach is to always reset when asked to connect again.
	if w.status == "connecting" {
		w.mu.Unlock()
		w.client.Disconnect()
		w.mu.Lock()
		w.status = "disconnected"
		w.qrCode = ""
	}

	// If already connected and healthy, return success
	if w.status == "connected" && w.client.IsConnected() {
		w.mu.Unlock()
		return nil
	}

	// Clear stale QR code and mark as connecting
	w.qrCode = ""
	w.status = "connecting"
	w.mu.Unlock()

	// Reset channels for fresh connection
	w.resetChannels()

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
				// Channel closed unexpectedly (e.g., client disconnected before timeout event)
				// Reset status so a new connection attempt can be made
				w.mu.Lock()
				if w.status == "connecting" {
					w.status = "disconnected"
					w.qrCode = ""
				}
				w.mu.Unlock()
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
		w.connectedAt = time.Now() // Track connection time to filter offline messages
		if w.client.Store.ID != nil {
			w.phone = w.client.Store.ID.User
		}
		webhookURL := w.webhookURL
		w.mu.Unlock()

		select {
		case w.ConnectedChan <- true:
		default:
		}

		// Send webhook event
		if webhookURL != "" {
			GetWebhookSender().SendEventAsync(webhookURL, WebhookEvent{
				Event:     "connection.connected",
				Instance:  w.instanceID,
				Timestamp: time.Now().Format(time.RFC3339),
				Data: StatusData{
					Status:      "connected",
					PhoneNumber: w.phone,
				},
			})
		}

	case *events.Disconnected:
		w.mu.Lock()
		w.status = "disconnected"
		w.qrCode = ""
		webhookURL := w.webhookURL
		w.mu.Unlock()

		// Send webhook event
		if webhookURL != "" {
			GetWebhookSender().SendEventAsync(webhookURL, WebhookEvent{
				Event:     "connection.disconnected",
				Instance:  w.instanceID,
				Timestamp: time.Now().Format(time.RFC3339),
				Data: StatusData{
					Status: "disconnected",
				},
			})
		}

	case *events.LoggedOut:
		w.mu.Lock()
		w.status = "disconnected"
		w.phone = ""
		w.qrCode = ""
		webhookURL := w.webhookURL
		w.mu.Unlock()

		// Send webhook event
		if webhookURL != "" {
			GetWebhookSender().SendEventAsync(webhookURL, WebhookEvent{
				Event:     "connection.logged_out",
				Instance:  w.instanceID,
				Timestamp: time.Now().Format(time.RFC3339),
				Data: StatusData{
					Status: "logged_out",
					Reason: "Logged out from another device",
				},
			})
		}

	case *events.PairSuccess:
		w.mu.Lock()
		w.status = "connected"
		w.phone = v.ID.User
		w.qrCode = ""
		w.mu.Unlock()

	case *events.StreamError:
		w.ErrorChan <- fmt.Errorf("stream error: %s", v.Code)

	case *events.Message:
		w.handleIncomingMessage(v)
	}
}

// handleIncomingMessage processes incoming messages and sends to webhook
func (w *WAClient) handleIncomingMessage(msg *events.Message) {
	w.mu.RLock()
	webhookURL := w.webhookURL
	ignoreGroups := w.ignoreGroups
	connectedAt := w.connectedAt
	w.mu.RUnlock()

	// Skip if no webhook configured
	if webhookURL == "" {
		return
	}

	// Ignore messages from newsletters/channels
	if msg.Info.Chat.Server == "newsletter" {
		return
	}

	// Ignore offline/historical messages: only process messages received after connection
	// This prevents the webhook from receiving messages that were pending while the server was offline
	if !connectedAt.IsZero() && msg.Info.Timestamp.Before(connectedAt) {
		return
	}

	// Check if it's a group message
	isGroup := msg.Info.IsGroup
	if ignoreGroups && isGroup {
		return
	}

	// Build message data
	msgData := MessageData{
		From:      msg.Info.Sender.User,
		MessageID: msg.Info.ID,
		IsGroup:   isGroup,
		Timestamp: msg.Info.Timestamp.Unix(),
	}

	// Get sender name from contact store if available
	if msg.Info.PushName != "" {
		msgData.FromName = msg.Info.PushName
	}

	// Set group info if applicable
	if isGroup {
		msgData.GroupID = msg.Info.Chat.User
		// Try to get group name from store
		if groupInfo, err := w.client.GetGroupInfo(context.Background(), msg.Info.Chat); err == nil {
			msgData.GroupName = groupInfo.Name
		}
	}

	// Determine message type and content
	message := msg.Message
	if message == nil {
		return
	}

	if message.Conversation != nil {
		msgData.MessageType = "text"
		msgData.Text = *message.Conversation
	} else if message.ExtendedTextMessage != nil {
		msgData.MessageType = "text"
		if message.ExtendedTextMessage.Text != nil {
			msgData.Text = *message.ExtendedTextMessage.Text
		}
	} else if message.ImageMessage != nil {
		msgData.MessageType = "image"
		if message.ImageMessage.Caption != nil {
			msgData.Caption = *message.ImageMessage.Caption
		}
		if message.ImageMessage.Mimetype != nil {
			msgData.MimeType = *message.ImageMessage.Mimetype
		}
		// Download and encode image as base64
		if data, err := w.client.Download(context.Background(), message.ImageMessage); err == nil {
			msgData.MediaBase64 = base64.StdEncoding.EncodeToString(data)
		}
	} else if message.VideoMessage != nil {
		msgData.MessageType = "video"
		if message.VideoMessage.Caption != nil {
			msgData.Caption = *message.VideoMessage.Caption
		}
		if message.VideoMessage.Mimetype != nil {
			msgData.MimeType = *message.VideoMessage.Mimetype
		}
		// Download and encode video as base64
		if data, err := w.client.Download(context.Background(), message.VideoMessage); err == nil {
			msgData.MediaBase64 = base64.StdEncoding.EncodeToString(data)
		}
	} else if message.AudioMessage != nil {
		msgData.MessageType = "audio"
		if message.AudioMessage.Mimetype != nil {
			msgData.MimeType = *message.AudioMessage.Mimetype
		}
		// Download and encode audio as base64
		if data, err := w.client.Download(context.Background(), message.AudioMessage); err == nil {
			msgData.MediaBase64 = base64.StdEncoding.EncodeToString(data)
		}
	} else if message.DocumentMessage != nil {
		msgData.MessageType = "document"
		if message.DocumentMessage.Caption != nil {
			msgData.Caption = *message.DocumentMessage.Caption
		}
		if message.DocumentMessage.Mimetype != nil {
			msgData.MimeType = *message.DocumentMessage.Mimetype
		}
		if message.DocumentMessage.FileName != nil {
			msgData.FileName = *message.DocumentMessage.FileName
		}
		// Download and encode document as base64
		if data, err := w.client.Download(context.Background(), message.DocumentMessage); err == nil {
			msgData.MediaBase64 = base64.StdEncoding.EncodeToString(data)
		}
	} else if message.StickerMessage != nil {
		msgData.MessageType = "sticker"
		if message.StickerMessage.Mimetype != nil {
			msgData.MimeType = *message.StickerMessage.Mimetype
		}
		// Download and encode sticker as base64
		if data, err := w.client.Download(context.Background(), message.StickerMessage); err == nil {
			msgData.MediaBase64 = base64.StdEncoding.EncodeToString(data)
		}
	} else if message.ContactMessage != nil {
		msgData.MessageType = "contact"
		if message.ContactMessage.DisplayName != nil {
			msgData.Text = *message.ContactMessage.DisplayName
		}
	} else if message.LocationMessage != nil {
		msgData.MessageType = "location"
	} else {
		msgData.MessageType = "unknown"
	}

	// Send to webhook asynchronously
	GetWebhookSender().SendEventAsync(webhookURL, WebhookEvent{
		Event:     "message.received",
		Instance:  w.instanceID,
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      msgData,
	})
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

// resetChannels resets the event channels for a fresh connection attempt
func (w *WAClient) resetChannels() {
	// Drain existing channels (non-blocking)
	for {
		select {
		case <-w.QRCodeChan:
		default:
			goto drainConnected
		}
	}
drainConnected:
	for {
		select {
		case <-w.ConnectedChan:
		default:
			goto drainError
		}
	}
drainError:
	for {
		select {
		case <-w.ErrorChan:
		default:
			goto donedraining
		}
	}
donedraining:
	// Recreate channels with fresh buffers
	w.QRCodeChan = make(chan string, 10)
	w.ConnectedChan = make(chan bool, 1)
	w.ErrorChan = make(chan error, 10)
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

// SetWebhook sets the webhook URL for this client
func (w *WAClient) SetWebhook(url string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.webhookURL = url
}

// SetIgnoreGroups sets whether to ignore group messages
func (w *WAClient) SetIgnoreGroups(ignore bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ignoreGroups = ignore
}

// GetWebhookURL returns the current webhook URL
func (w *WAClient) GetWebhookURL() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.webhookURL
}

// SetProxy sets the proxy URL for this client
func (w *WAClient) SetProxy(url string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.client.SetProxyAddress(url)
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

// SendResponse contains the response after sending a message
type SendResponse struct {
	MessageID string
	Timestamp time.Time
}

// ContactInfo contains contact information
type ContactInfo struct {
	JID          string `json:"jid"`
	PhoneNumber  string `json:"phoneNumber"`
	Name         string `json:"name"`
	PushName     string `json:"pushName"`
	BusinessName string `json:"businessName"`
}

// parseRecipientJID correctly parses a recipient string into a JID
func (w *WAClient) parseRecipientJID(recipient string) (types.JID, error) {
	if recipient == "" {
		return types.EmptyJID, fmt.Errorf("recipient is empty")
	}

	// If it's already a full JID, parse it directly
	if strings.Contains(recipient, "@") {
		return types.ParseJID(recipient)
	}

	// Otherwise, assume it's a phone number and add the server
	return types.NewJID(recipient, types.DefaultUserServer), nil
}

// validateNumber checks if a JID is a valid WhatsApp account
func (w *WAClient) validateNumber(ctx context.Context, jid types.JID) error {
	// Only validate user JIDs (not groups or newsletters)
	if jid.Server != types.DefaultUserServer {
		return nil
	}

	resp, err := w.client.IsOnWhatsApp(ctx, []string{jid.User})
	if err != nil {
		return fmt.Errorf("failed to check if number is on WhatsApp: %w", err)
	}

	if len(resp) == 0 || !resp[0].IsIn {
		return fmt.Errorf("number is not on WhatsApp")
	}

	return nil
}

// SendTextMessage sends a text message with optional typing indicator
func (w *WAClient) SendTextMessage(ctx context.Context, recipient string, text string, simulateTyping bool, typingDurationMs int) (*SendResponse, error) {
	if !w.client.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	// Parse recipient JID
	recipientJID, err := w.parseRecipientJID(recipient)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient: %w", err)
	}

	// Validate number
	if err := w.validateNumber(ctx, recipientJID); err != nil {
		return nil, err
	}

	// Simulate typing if requested
	if simulateTyping {
		duration := typingDurationMs
		if duration <= 0 {
			duration = 2000 // Default 2 seconds
		}

		// Send typing presence
		err := w.client.SendChatPresence(ctx, recipientJID, types.ChatPresenceComposing, types.ChatPresenceMediaText)
		if err != nil {
			// Log but don't fail the message send
			fmt.Printf("Warning: failed to send typing presence: %v\n", err)
		}

		// Wait for typing duration
		time.Sleep(time.Duration(duration) * time.Millisecond)

		// Clear typing presence
		w.client.SendChatPresence(ctx, recipientJID, types.ChatPresencePaused, types.ChatPresenceMediaText)
	}

	// Create and send message
	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	resp, err := w.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &SendResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendImageMessage sends an image with optional caption
func (w *WAClient) SendImageMessage(ctx context.Context, recipient string, imageData []byte, mimeType string, caption string) (*SendResponse, error) {
	if !w.client.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	// Parse recipient JID
	recipientJID, err := w.parseRecipientJID(recipient)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient: %w", err)
	}

	// Validate number
	if err := w.validateNumber(ctx, recipientJID); err != nil {
		return nil, err
	}

	// Upload image
	uploadResp, err := w.client.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Create image message
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
			Caption:       proto.String(caption),
		},
	}

	resp, err := w.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send image: %w", err)
	}

	return &SendResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendVideoMessage sends a video with optional caption
func (w *WAClient) SendVideoMessage(ctx context.Context, recipient string, videoData []byte, mimeType string, caption string) (*SendResponse, error) {
	if !w.client.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	// Parse recipient JID
	recipientJID, err := w.parseRecipientJID(recipient)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient: %w", err)
	}

	// Validate number
	if err := w.validateNumber(ctx, recipientJID); err != nil {
		return nil, err
	}

	// Upload video
	uploadResp, err := w.client.Upload(ctx, videoData, whatsmeow.MediaVideo)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	// Create video message
	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(videoData))),
			Caption:       proto.String(caption),
		},
	}

	resp, err := w.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send video: %w", err)
	}

	return &SendResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendAudioMessage sends an audio file with optional recording indicator
func (w *WAClient) SendAudioMessage(ctx context.Context, recipient string, audioData []byte, mimeType string, simulateRecording bool, recordingDurationMs int, ptt bool) (*SendResponse, error) {
	if !w.client.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	// Parse recipient JID
	recipientJID, err := w.parseRecipientJID(recipient)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient: %w", err)
	}

	// Validate number
	if err := w.validateNumber(ctx, recipientJID); err != nil {
		return nil, err
	}

	// Simulate recording if requested
	if simulateRecording {
		duration := recordingDurationMs
		if duration <= 0 {
			duration = 3000 // Default 3 seconds
		}

		// Send recording presence
		err := w.client.SendChatPresence(ctx, recipientJID, types.ChatPresenceComposing, types.ChatPresenceMediaAudio)
		if err != nil {
			fmt.Printf("Warning: failed to send recording presence: %v\n", err)
		}

		// Wait for recording duration
		time.Sleep(time.Duration(duration) * time.Millisecond)

		// Clear recording presence
		w.client.SendChatPresence(ctx, recipientJID, types.ChatPresencePaused, types.ChatPresenceMediaAudio)
	}

	// Upload audio
	uploadResp, err := w.client.Upload(ctx, audioData, whatsmeow.MediaAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio: %w", err)
	}

	// Create audio message
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(audioData))),
			PTT:           proto.Bool(ptt),
		},
	}

	resp, err := w.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send audio: %w", err)
	}

	return &SendResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// SendDocumentMessage sends a document with optional caption and filename
func (w *WAClient) SendDocumentMessage(ctx context.Context, recipient string, docData []byte, mimeType string, fileName string, caption string) (*SendResponse, error) {
	if !w.client.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	// Parse recipient JID
	recipientJID, err := w.parseRecipientJID(recipient)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient: %w", err)
	}

	// Validate number
	if err := w.validateNumber(ctx, recipientJID); err != nil {
		return nil, err
	}

	// Upload document
	uploadResp, err := w.client.Upload(ctx, docData, whatsmeow.MediaDocument)
	if err != nil {
		return nil, fmt.Errorf("failed to upload document: %w", err)
	}

	// Create document message
	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(docData))),
			FileName:      proto.String(fileName),
			Caption:       proto.String(caption),
		},
	}

	resp, err := w.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send document: %w", err)
	}

	return &SendResponse{
		MessageID: resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// GetContacts retrieves all contacts from the store (excluding groups)
func (w *WAClient) GetContacts(ctx context.Context) ([]ContactInfo, error) {
	if !w.client.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	// Get all contacts from the store
	contacts, err := w.client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	var result []ContactInfo
	for jid, contact := range contacts {
		// Filter out groups - only include user JIDs
		if jid.Server != types.DefaultUserServer {
			continue
		}

		result = append(result, ContactInfo{
			JID:          jid.String(),
			PhoneNumber:  jid.User,
			Name:         contact.FullName,
			PushName:     contact.PushName,
			BusinessName: contact.BusinessName,
		})
	}

	return result, nil
}

// SetPresence sets the online/offline presence for this instance
func (w *WAClient) SetPresence(available bool) error {
	if !w.client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}
	if available {
		return w.client.SendPresence(context.Background(), types.PresenceAvailable)
	}
	return w.client.SendPresence(context.Background(), types.PresenceUnavailable)
}
