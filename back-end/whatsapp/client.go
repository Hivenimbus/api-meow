package whatsapp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	db "api-meow/internal/db"
)

// isOggOpus reports whether the MIME type is OGG/Opus (required for WhatsApp PTT).
func isOggOpus(mimeType string) bool {
	return strings.Contains(mimeType, "ogg")
}

// calcOggDurationSeconds extracts the total duration from an OGG/Opus stream.
// Reads the pre_skip from the OpusHead (first audio page payload) and subtracts
// it from the max granule position. Opus uses 48000 Hz sample rate.
func calcOggDurationSeconds(data []byte) uint32 {
	magic := []byte("OggS")
	var maxGranule int64
	var preSkip int64

	for i := 0; i < len(data)-27; {
		idx := bytes.Index(data[i:], magic)
		if idx < 0 {
			break
		}
		pageStart := i + idx

		if pageStart+27 > len(data) {
			break
		}

		headerType := data[pageStart+5]
		granule := int64(binary.LittleEndian.Uint64(data[pageStart+6 : pageStart+14]))

		segCount := int(data[pageStart+26])
		headerEnd := pageStart + 27 + segCount
		if headerEnd > len(data) {
			break
		}
		var dataSize int
		for j := 0; j < segCount; j++ {
			dataSize += int(data[pageStart+27+j])
		}
		payloadStart := headerEnd
		payloadEnd := payloadStart + dataSize

		// BOS page: if payload starts with "OpusHead", read pre_skip (bytes 10-11 of payload)
		if headerType&0x02 != 0 && payloadEnd <= len(data) && payloadEnd-payloadStart >= 12 {
			payload := data[payloadStart:payloadEnd]
			if len(payload) >= 12 && string(payload[:8]) == "OpusHead" {
				preSkip = int64(binary.LittleEndian.Uint16(payload[10:12]))
			}
		}

		if granule > 0 && granule != -1 && granule > maxGranule {
			maxGranule = granule
		}

		i = headerEnd + dataSize
	}

	if maxGranule > preSkip {
		secs := uint32((maxGranule - preSkip) / 48000)
		if secs == 0 {
			secs = 1
		}
		return secs
	}
	return 1
}

// WAClient wraps a whatsmeow client with additional functionality
type WAClient struct {
	instanceID   string
	client       *whatsmeow.Client
	db           *gorm.DB
	qrCode       string
	status       string // disconnected, connecting, connected
	phone        string
	webhookURL   string
	ignoreGroups bool
	connectedAt          time.Time // Track when client connected to filter offline messages
	pairedAt             time.Time // Track QR pairing time to suppress brief disconnect during session setup
	historySyncProgress  int32     // atomic: 0-100, updated by HistorySync events
	mu                   sync.RWMutex

	// Contacts cache — invalidated by HistorySync events and new Connect() calls
	contactsCache      []ContactInfo
	contactsCacheAt    time.Time
	contactsCacheDirty bool

	// Auto-reconnect
	reconnectFn   func()
	autoReconnect bool
	reconnecting  int32 // atomic: 1 = reconnect goroutine already in flight

	// Connection guard: prevents two concurrent Connect() calls from both reaching client.Connect()
	isConnecting int32 // atomic: 1 = connect in progress

	// Set to 1 during graceful shutdown so the Disconnected event handler skips the DB
	// status update, keeping "connected" in the DB for the next container to restore.
	shuttingDown int32 // atomic

	// Set to 1 when the session is replaced by another connection (rolling deploy or
	// another device connecting). Prevents the old connection from fighting back and
	// kicking the new connection in a reconnect loop.
	streamReplaced int32 // atomic

	// Limits concurrent processHistorySync goroutines to avoid DB contention during initial sync
	historySyncSem chan struct{}

	// Channels for events
	QRCodeChan    chan string
	ConnectedChan chan bool
	ErrorChan     chan error
}

// SetReconnectFunc sets the function to call when the connection drops unexpectedly.
// Auto-reconnect will be enabled once the client successfully connects.
func (w *WAClient) SetReconnectFunc(fn func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.reconnectFn = fn
}

// SetShuttingDown marks the client as being closed due to a graceful shutdown.
// This prevents the Disconnected event handler from writing "disconnected" to the DB,
// so the next container can find the instance and reconnect it.
func (w *WAClient) SetShuttingDown() {
	atomic.StoreInt32(&w.shuttingDown, 1)
}

// DisableAutoReconnect prevents the client from reconnecting after a disconnect.
// Call this before an intentional disconnect.
func (w *WAClient) DisableAutoReconnect() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.autoReconnect = false
}

// NewWAClient creates a new WhatsApp client wrapper
func NewWAClient(instanceID string, client *whatsmeow.Client, db *gorm.DB) *WAClient {
	wac := &WAClient{
		instanceID:     instanceID,
		client:         client,
		db:             db,
		status:         "disconnected",
		QRCodeChan:     make(chan string, 10),
		ConnectedChan:  make(chan bool, 1),
		ErrorChan:      make(chan error, 10),
		historySyncSem: make(chan struct{}, 3),
	}

	// Register event handler
	client.AddEventHandler(wac.eventHandler)

	return wac
}

// Connect initiates the connection to WhatsApp
func (w *WAClient) Connect(ctx context.Context) error {
	w.mu.Lock()

	// If status is "connecting", behavior depends on whether the device is already registered:
	// - Store.ID != nil (already logged in): connection is in progress from another goroutine,
	//   return nil to avoid killing it (prevents race condition on server restart).
	// - Store.ID == nil (QR scan in progress): reset to generate a fresh QR code.
	if w.status == "connecting" {
		if w.client.Store.ID != nil {
			// Already logged in — don't interfere with ongoing connection
			w.mu.Unlock()
			return nil
		}
		// Not yet logged in (QR scan) — reset for a fresh QR
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

	// Guard: only one goroutine proceeds to the actual client.Connect() call.
	// Others see status="connecting" on the next iteration and bail out above.
	if !atomic.CompareAndSwapInt32(&w.isConnecting, 0, 1) {
		return nil
	}
	defer atomic.StoreInt32(&w.isConnecting, 0)

	// Reset history sync progress and contacts cache for new connection
	atomic.StoreInt32(&w.historySyncProgress, 0)
	w.mu.Lock()
	w.contactsCache = nil
	w.contactsCacheDirty = false
	w.mu.Unlock()

	// Reset channels for fresh connection
	w.resetChannels()

	// Check if already logged in
	if w.client.Store.ID != nil {
		// Already registered, just connect
		err := w.client.Connect()
		if err != nil {
			if strings.Contains(err.Error(), "websocket is already connected") {
				return nil // socket already open from a concurrent connect — treat as success
			}
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
		if strings.Contains(err.Error(), "websocket is already connected") {
			return nil // socket already open from a concurrent connect — treat as success
		}
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
		atomic.StoreInt32(&w.reconnecting, 0)   // connection succeeded — clear reconnect guard
		atomic.StoreInt32(&w.streamReplaced, 0) // clear replaced flag in case we reconnected after replacement
		w.mu.Lock()
		w.status = "connected"
		w.connectedAt = time.Now() // Track connection time to filter offline messages
		w.pairedAt = time.Time{}   // Clear pairing flag — fully connected now
		if w.client.Store.ID != nil {
			w.phone = w.client.Store.ID.User
		}
		phone := w.phone
		webhookURL := w.webhookURL
		if w.reconnectFn != nil {
			w.autoReconnect = true
		}
		w.mu.Unlock()

		// Persist connected status so the next container restart can restore this session.
		// This also covers the auto-reconnect case where status was "disconnected" in DB.
		if w.db != nil && phone != "" {
			w.db.Model(&db.Instance{}).Where("name = ?", w.instanceID).Updates(map[string]interface{}{
				"status":       "connected",
				"phone_number": phone,
			})
		}

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
		// Ignore brief disconnect that whatsmeow fires between PairSuccess and Connected
		// during QR pairing session establishment (typically lasts < 5 seconds).
		pairedAt := w.pairedAt
		if !pairedAt.IsZero() && time.Since(pairedAt) < 15*time.Second {
			w.mu.Unlock()
			return
		}
		w.status = "disconnected"
		w.qrCode = ""
		webhookURL := w.webhookURL
		shouldReconnect := w.autoReconnect
		reconnectFn := w.reconnectFn
		w.mu.Unlock()

		// Persist disconnected status to DB — but skip during graceful shutdown so the
		// next container can find the instance (status stays "connected") and reconnect it.
		if w.db != nil && atomic.LoadInt32(&w.shuttingDown) == 0 {
			w.db.Model(&db.Instance{}).Where("name = ?", w.instanceID).Updates(map[string]interface{}{
				"status": "disconnected",
			})
		}

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

		// Don't reconnect during graceful shutdown or when the session was intentionally
		// replaced by another connection (e.g. rolling deploy starting a new container).
		// Without these guards the old container fights the new one in a reconnect loop.
		if shouldReconnect && reconnectFn != nil &&
			atomic.LoadInt32(&w.shuttingDown) == 0 &&
			atomic.LoadInt32(&w.streamReplaced) == 0 {
			if atomic.CompareAndSwapInt32(&w.reconnecting, 0, 1) {
				go func() {
					defer atomic.StoreInt32(&w.reconnecting, 0)
					reconnectFn()
				}()
			}
		}

	case *events.LoggedOut:
		w.mu.Lock()
		w.status = "disconnected"
		w.phone = ""
		w.qrCode = ""
		w.autoReconnect = false
		webhookURL := w.webhookURL
		w.mu.Unlock()

		// Persist logged out status to DB and clear phone number
		if w.db != nil {
			w.db.Model(&db.Instance{}).Where("name = ?", w.instanceID).Updates(map[string]interface{}{
				"status":       "disconnected",
				"phone_number": nil,
			})
		}

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
		w.pairedAt = time.Now() // Mark pairing time to suppress brief disconnect during session setup
		w.mu.Unlock()

	case *events.StreamError:
		if v.Code == "replaced" {
			// Session was taken over by another connection (rolling deploy / another device).
			// Mark the flag so the subsequent Disconnected event won't trigger a reconnect.
			atomic.StoreInt32(&w.streamReplaced, 1)
		}
		w.ErrorChan <- fmt.Errorf("stream error: %s", v.Code)

	case *events.HistorySync:
		go func(ev *events.HistorySync) {
			w.historySyncSem <- struct{}{}
			defer func() { <-w.historySyncSem }()
			w.processHistorySync(ev)
		}(v)

	case *events.Message:
		go w.saveInteractionJID(v)
		go w.handleIncomingMessage(v)
	}
}

// handleIncomingMessage processes incoming messages and sends to webhook
func (w *WAClient) handleIncomingMessage(msg *events.Message) {
	w.mu.RLock()
	webhookURL := w.webhookURL
	ignoreGroups := w.ignoreGroups
	connectedAt := w.connectedAt
	w.mu.RUnlock()

	log.Printf("[Webhook] Message received for instance %s: from=%s type=%s webhookURL=%q",
		w.instanceID, msg.Info.Sender.User, msg.Info.Chat.Server, webhookURL)

	// Skip if no webhook configured
	if webhookURL == "" {
		log.Printf("[Webhook] Skipping: no webhook URL configured for instance %s", w.instanceID)
		return
	}

	// Ignore messages from newsletters/channels
	if msg.Info.Chat.Server == "newsletter" {
		return
	}

	// Ignore offline/historical messages: only process messages received after connection.
	// Use whole-second precision for connectedAt to avoid filtering messages sent in the
	// same second as the connection (WhatsApp timestamps have 1-second resolution).
	if !connectedAt.IsZero() && msg.Info.Timestamp.Before(connectedAt.Truncate(time.Second)) {
		log.Printf("[Webhook] Skipping offline message for instance %s: msg_ts=%v connected_at=%v",
			w.instanceID, msg.Info.Timestamp, connectedAt)
		return
	}

	// Check if it's a group message
	isGroup := msg.Info.IsGroup
	if ignoreGroups && isGroup {
		return
	}

	// For DMs, use Chat.User (always the phone number).
	// Sender.User may be a privacy LID on newer WhatsApp multi-device accounts.
	from := msg.Info.Chat.User
	if isGroup {
		from = msg.Info.Sender.User
	}

	// Build message data
	msgData := MessageData{
		From:      from,
		MessageID: msg.Info.ID,
		IsGroup:   isGroup,
		IsFromMe:  msg.Info.IsFromMe,
		Timestamp: msg.Info.Timestamp.Unix(),
	}

	// Get sender name from contact store if available
	if msg.Info.PushName != "" {
		msgData.FromName = msg.Info.PushName
	}

	// Single context with timeout shared by all network operations in this handler
	// (GetGroupInfo + media downloads). Prevents goroutine leaks if CDN hangs.
	mediaCtx, mediaCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer mediaCancel()

	// Resolve LID to real phone number (DMs and group senders).
	// On newer multi-device accounts, JIDs may be @lid privacy identifiers, not phone numbers.
	{
		jidToResolve := msg.Info.Chat.String() // for DMs, Chat is the contact's JID
		if isGroup {
			jidToResolve = msg.Info.Sender.String() // for groups, Sender is the member's JID
		}
		if resolved, _ := w.resolveToPhone(mediaCtx, jidToResolve); resolved != "" {
			from = resolved
			msgData.From = resolved
		}
	}

	// Set group info if applicable
	if isGroup {
		msgData.GroupID = msg.Info.Chat.User
		// Try to get group name from store
		if groupInfo, err := w.client.GetGroupInfo(mediaCtx, msg.Info.Chat); err == nil {
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
		if data, err := w.client.Download(mediaCtx, message.ImageMessage); err == nil {
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
		if data, err := w.client.Download(mediaCtx, message.VideoMessage); err == nil {
			msgData.MediaBase64 = base64.StdEncoding.EncodeToString(data)
		}
	} else if message.AudioMessage != nil {
		msgData.MessageType = "audio"
		if message.AudioMessage.Mimetype != nil {
			msgData.MimeType = *message.AudioMessage.Mimetype
		}
		// Download and encode audio as base64
		if data, err := w.client.Download(mediaCtx, message.AudioMessage); err == nil {
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
		if data, err := w.client.Download(mediaCtx, message.DocumentMessage); err == nil {
			msgData.MediaBase64 = base64.StdEncoding.EncodeToString(data)
		}
	} else if message.StickerMessage != nil {
		msgData.MessageType = "sticker"
		if message.StickerMessage.Mimetype != nil {
			msgData.MimeType = *message.StickerMessage.Mimetype
		}
		// Download and encode sticker as base64
		if data, err := w.client.Download(mediaCtx, message.StickerMessage); err == nil {
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
	status := w.status
	phone := w.phone
	pairedAt := w.pairedAt
	w.mu.RUnlock()

	// If cached status says connected but socket is down, check if we're in the
	// brief reconnect window after QR pairing (whatsmeow disconnects and reconnects
	// between PairSuccess and Connected). During this window, report "connecting"
	// instead of "disconnected" to prevent false "QR Code expired" errors.
	// Do NOT mutate w.status here — state changes must only come from event handlers.
	if status == "connected" && !w.client.IsConnected() {
		if !pairedAt.IsZero() && time.Since(pairedAt) < 15*time.Second {
			return "connecting", phone
		}
		return "disconnected", phone
	}

	return status, phone
}

// GetSyncProgress returns the history sync progress (0-100). 100 means sync is complete.
func (w *WAClient) GetSyncProgress() int {
	return int(atomic.LoadInt32(&w.historySyncProgress))
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

	// PTT (voice message) requires ogg/opus. If the audio is already ogg/opus
	// (converted upstream by the Nuxt server), send as PTT. Otherwise send as
	// regular audio — no local ffmpeg conversion needed.
	uploadData := audioData
	uploadMime := mimeType
	if !isOggOpus(mimeType) {
		// Not ogg/opus — cannot send as PTT; send as regular audio instead
		log.Printf("[Audio] mimeType %q is not ogg/opus, sending as regular audio (non-PTT)", mimeType)
		ptt = false
	} else {
		ptt = true
	}

	// Upload audio
	uploadResp, err := w.client.Upload(ctx, uploadData, whatsmeow.MediaAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio: %w", err)
	}

	// Calculate duration (Seconds field is required for PTT display)
	var seconds uint32 = 1
	if strings.Contains(uploadMime, "ogg") {
		seconds = calcOggDurationSeconds(uploadData)
	}

	// Create audio message
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(uploadMime),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(uploadData))),
			PTT:           proto.Bool(ptt),
			Seconds:       proto.Uint32(seconds),
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

// processHistorySync saves conversation JIDs from the WhatsApp history sync event
func (w *WAClient) processHistorySync(v *events.HistorySync) {
	if w.db == nil || w.client.Store.ID == nil {
		return
	}
	ourJID := w.client.Store.ID.String()

	for _, conv := range v.Data.GetConversations() {
		jidStr := conv.GetID()
		if jidStr == "" || strings.Contains(jidStr, "@g.us") || strings.Contains(jidStr, "@broadcast") {
			continue
		}

		displayName := conv.GetDisplayName()
		if displayName == "" {
			displayName = conv.GetName()
		}

		w.db.Exec(`
			INSERT INTO chat_jids (instance_jid, jid, name)
			VALUES (?, ?, ?)
			ON CONFLICT (instance_jid, jid) DO UPDATE SET name = EXCLUDED.name WHERE EXCLUDED.name != ''`,
			ourJID, jidStr, displayName)

		// Also capture senders from individual messages
		for _, histMsg := range conv.GetMessages() {
			if histMsg.GetMessage() == nil || histMsg.GetMessage().GetKey() == nil {
				continue
			}
			msgJID := histMsg.GetMessage().GetKey().GetRemoteJID()
			if msgJID == "" || strings.Contains(msgJID, "@g.us") || strings.Contains(msgJID, "@broadcast") {
				continue
			}
			w.db.Exec(`
				INSERT INTO chat_jids (instance_jid, jid, name)
				VALUES (?, ?, ?)
				ON CONFLICT (instance_jid, jid) DO NOTHING`,
				ourJID, msgJID, "")
		}
	}

	// Update history sync progress (0-100) and mark contacts cache as dirty
	atomic.StoreInt32(&w.historySyncProgress, int32(v.Data.GetProgress()))
	w.mu.Lock()
	w.contactsCacheDirty = true
	w.mu.Unlock()
}

// saveInteractionJID saves a contact JID from real-time message interactions
func (w *WAClient) saveInteractionJID(v *events.Message) {
	if w.db == nil || w.client.Store.ID == nil {
		return
	}
	jidStr := v.Info.Chat.String()
	if strings.Contains(jidStr, "@g.us") || strings.Contains(jidStr, "@broadcast") || strings.Contains(jidStr, "@newsletter") {
		return
	}
	ourJID := w.client.Store.ID.String()
	pushName := v.Info.PushName

	w.db.Exec(`
		INSERT INTO chat_jids (instance_jid, jid, push_name)
		VALUES (?, ?, ?)
		ON CONFLICT (instance_jid, jid) DO UPDATE SET push_name = EXCLUDED.push_name WHERE EXCLUDED.push_name != ''`,
		ourJID, jidStr, pushName)
}

// resolveToPhone resolves any JID string (regular @s.whatsapp.net or LID @lid) to a phone number.
// Returns "" if the JID cannot be mapped to a phone number.
func (w *WAClient) resolveToPhone(ctx context.Context, jidStr string) (phone string, canonicalJID string) {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return "", ""
	}
	if jid.Server == types.DefaultUserServer {
		return jid.User, jidStr
	}
	// LID (Linked Identity) — use whatsmeow's built-in LID store (has in-memory cache)
	// whatsmeow_lid_map stores only the user parts (e.g. "12345678" not "12345678@lid")
	// so we must use the native API instead of raw SQL to avoid format mismatch.
	if jid.Server == types.HiddenUserServer && w.client.Store != nil {
		pnJID, err := w.client.Store.LIDs.GetPNForLID(ctx, jid.ToNonAD())
		if err == nil && !pnJID.IsEmpty() && pnJID.Server == types.DefaultUserServer {
			return pnJID.User, pnJID.User + "@" + types.DefaultUserServer
		}
	}
	return "", ""
}

// FetchContacts retrieves contacts from 5 sources, deduplicates, and applies anti-spam filters.
// Handles modern WhatsApp LID (Linked Identity) JIDs via whatsmeow_lid_map.
// Results are cached in memory for up to 5 minutes; cache is invalidated by HistorySync events.
func (w *WAClient) FetchContacts(ctx context.Context) ([]ContactInfo, error) {
	// Serve from cache if fresh and not dirty
	w.mu.RLock()
	if !w.contactsCacheDirty && len(w.contactsCache) > 0 && time.Since(w.contactsCacheAt) < 5*time.Minute {
		result := make([]ContactInfo, len(w.contactsCache))
		copy(result, w.contactsCache)
		w.mu.RUnlock()
		return result, nil
	}
	w.mu.RUnlock()

	seen := make(map[string]*ContactInfo) // keyed by phone number

	// Pre-load all LID→PN mappings into in-memory cache in one bulk query.
	// FillCache is not on the LIDStore interface, so use a type assertion.
	// Without this, resolveToPhone makes one DB query per LID contact (very slow).
	if w.client.Store != nil && w.client.Store.LIDs != nil {
		type lidCacheFiller interface {
			FillCache(ctx context.Context) error
		}
		if filler, ok := w.client.Store.LIDs.(lidCacheFiller); ok {
			_ = filler.FillCache(ctx)
		}
	}

	// Source 1 & 3: whatsmeow_contacts + Memory Store (GetAllContacts covers both)
	if w.client.Store.ID != nil {
		contacts, err := w.client.Store.Contacts.GetAllContacts(ctx)
		if err == nil {
			for jid, contact := range contacts {
				phone, canonicalJID := w.resolveToPhone(ctx, jid.String())
				if phone == "" {
					continue
				}
				name := bestName(contact.FullName, contact.BusinessName, contact.PushName)
				seen[phone] = &ContactInfo{
					JID:          canonicalJID,
					PhoneNumber:  phone,
					Name:         name,
					PushName:     contact.PushName,
					BusinessName: contact.BusinessName,
				}
			}
		}
	}

	if w.db != nil && w.client.Store.ID != nil {
		ourJID := w.client.Store.ID.String()

		// Source 2: chat_jids (history sync data)
		type chatJIDRow struct {
			Jid      string
			Name     string
			PushName string
		}
		var chatRows []chatJIDRow
		w.db.Raw(`SELECT jid, name, push_name FROM chat_jids WHERE instance_jid = ?`, ourJID).Scan(&chatRows)
		for _, row := range chatRows {
			phone, canonicalJID := w.resolveToPhone(ctx, row.Jid)
			if phone == "" {
				continue
			}
			name := bestName(row.Name, "", row.PushName)
			if existing, exists := seen[phone]; !exists {
				seen[phone] = &ContactInfo{
					JID:         canonicalJID,
					PhoneNumber: phone,
					Name:        name,
					PushName:    row.PushName,
				}
			} else if existing.Name == "" && existing.PushName == "" {
				if name != "" {
					existing.Name = name
				}
				if row.PushName != "" {
					existing.PushName = row.PushName
				}
			}
		}

		// Source 4: whatsmeow_chat_settings (chats with any settings configured)
		type chatSettingsRow struct {
			ChatJid string
		}
		var settingsRows []chatSettingsRow
		w.db.Raw(`SELECT chat_jid FROM whatsmeow_chat_settings WHERE our_jid = ? AND chat_jid NOT LIKE '%@g.us' AND chat_jid NOT LIKE '%@broadcast'`, ourJID).Scan(&settingsRows)
		for _, row := range settingsRows {
			phone, canonicalJID := w.resolveToPhone(ctx, row.ChatJid)
			if phone == "" {
				continue
			}
			if _, exists := seen[phone]; !exists {
				seen[phone] = &ContactInfo{
					JID:         canonicalJID,
					PhoneNumber: phone,
				}
			}
		}

		// Source 5: whatsmeow_message_secrets (every chat where encrypted messages were exchanged)
		type msgSecretRow struct {
			ChatJid   string
			SenderJid string
		}
		var secretRows []msgSecretRow
		w.db.Raw(`SELECT DISTINCT chat_jid, sender_jid FROM whatsmeow_message_secrets
			WHERE our_jid = ?
			AND chat_jid NOT LIKE '%@g.us'
			AND chat_jid NOT LIKE '%@broadcast'
			AND chat_jid NOT LIKE '%@newsletter'
			ORDER BY rowid DESC LIMIT 10000`, ourJID).Scan(&secretRows)
		for _, row := range secretRows {
			for _, jidStr := range []string{row.ChatJid, row.SenderJid} {
				if jidStr == "" {
					continue
				}
				phone, canonicalJID := w.resolveToPhone(ctx, jidStr)
				if phone == "" {
					continue
				}
				if _, exists := seen[phone]; !exists {
					seen[phone] = &ContactInfo{
						JID:         canonicalJID,
						PhoneNumber: phone,
					}
				}
			}
		}
	}

	// Apply anti-spam filters and build result
	var result []ContactInfo
	for phone, info := range seen {
		// Determine effective name (prioritize: Name > PushName > "")
		name := info.Name
		if name == "" {
			name = info.PushName
		}
		// If name equals the phone number, treat as unnamed
		if name == phone || name == "+"+phone {
			name = ""
		}

		// Contacts without a saved name: apply filters
		if name == "" {
			// Discard numbers with 14+ digits (bots/spam/very long numbers)
			if len(phone) >= 14 {
				continue
			}
		}

		info.Name = name
		result = append(result, *info)
	}

	// Store in cache
	w.mu.Lock()
	w.contactsCache = result
	w.contactsCacheAt = time.Now()
	w.contactsCacheDirty = false
	w.mu.Unlock()

	return result, nil
}

// bestName returns the best available name from the provided candidates (priority order)
func bestName(fullName, businessName, pushName string) string {
	if fullName != "" {
		return fullName
	}
	if businessName != "" {
		return businessName
	}
	return pushName
}

// GetContacts retrieves contacts (calls FetchContacts for backwards compatibility)
func (w *WAClient) GetContacts(ctx context.Context) ([]ContactInfo, error) {
	return w.FetchContacts(ctx)
}

// GetProfilePictureURL returns the profile picture URL for the given phone number.
// Returns empty string (no error) if the contact has no picture or privacy settings block access.
func (w *WAClient) GetProfilePictureURL(ctx context.Context, phone string) (string, error) {
	if !w.client.IsConnected() {
		return "", fmt.Errorf("client is not connected")
	}
	jid, err := types.ParseJID(phone + "@s.whatsapp.net")
	if err != nil {
		return "", fmt.Errorf("invalid phone number: %w", err)
	}
	pic, err := w.client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{Preview: false})
	if err != nil || pic == nil {
		// Not critical — contact may have no photo or privacy settings block access
		return "", nil
	}
	return pic.URL, nil
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
