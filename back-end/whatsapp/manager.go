package whatsapp

import (
	"context"
	"fmt"
	"sync"

	_ "github.com/lib/pq" // Register PostgreSQL driver
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// InstanceManager manages multiple WhatsApp client instances
type InstanceManager struct {
	container *sqlstore.Container
	clients   map[string]*WAClient
	db        *gorm.DB
	mu        sync.RWMutex
	log       waLog.Logger
	ctx       context.Context
}

func init() {
	// Force WhatsApp to send full history sync on new device connections
	store.DeviceProps.RequireFullSync = proto.Bool(true)
}

// NewInstanceManager creates a new WhatsApp instance manager
func NewInstanceManager(dbURL string, db *gorm.DB) (*InstanceManager, error) {
	// Create logger
	log := waLog.Stdout("WhatsApp", "INFO", true)

	ctx := context.Background()

	// Create SQL store container using the database URL
	// Use "postgres" dialect for PostgreSQL
	container, err := sqlstore.New(ctx, "postgres", dbURL, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlstore container: %w", err)
	}

	// Upgrade database schema (creates whatsmeow tables if they don't exist)
	err = container.Upgrade(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade database schema: %w", err)
	}

	// Create chat_jids table for history sync and real-time interaction tracking
	if db != nil {
		if err := db.Exec(`
			CREATE TABLE IF NOT EXISTS chat_jids (
				instance_jid TEXT,
				jid          TEXT,
				name         TEXT NOT NULL DEFAULT '',
				push_name    TEXT NOT NULL DEFAULT '',
				PRIMARY KEY (instance_jid, jid)
			)`).Error; err != nil {
			log.Warnf("Failed to create chat_jids table: %v", err)
		}
		if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_jids_instance ON chat_jids (instance_jid)`).Error; err != nil {
			log.Warnf("Failed to create chat_jids index: %v", err)
		}
		if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_msg_secrets_our_jid ON whatsmeow_message_secrets(our_jid)`).Error; err != nil {
			log.Warnf("Failed to create msg_secrets index: %v", err)
		}
		if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_settings_our_jid ON whatsmeow_chat_settings(our_jid)`).Error; err != nil {
			log.Warnf("Failed to create chat_settings index: %v", err)
		}
	}

	log.Infof("WhatsApp store initialized successfully")

	manager := &InstanceManager{
		container: container,
		clients:   make(map[string]*WAClient),
		db:        db,
		log:       log,
		ctx:       ctx,
	}

	return manager, nil
}

// InstanceInfo contains info for restoring a client
type InstanceInfo struct {
	Name         string
	PhoneNumber  string
	WebhookURL   string
	IgnoreGroups bool
	ProxyURL     string
}

// RestoreClients restores clients from database using instance info from the app database
func (m *InstanceManager) RestoreClients(instances []InstanceInfo) {
	// Build a map of phone number -> instance info
	phoneToInfo := make(map[string]InstanceInfo)
	for _, inst := range instances {
		if inst.PhoneNumber != "" {
			phoneToInfo[inst.PhoneNumber] = inst
		}
	}

	devices, err := m.container.GetAllDevices(m.ctx)
	if err != nil {
		m.log.Errorf("Failed to get devices for restoration: %v", err)
		return
	}

	count := 0
	for _, device := range devices {
		if device.ID == nil {
			continue
		}

		// Get the phone number from the device JID
		phoneNumber := device.ID.User

		// Find the instance info for this phone number
		instanceInfo, found := phoneToInfo[phoneNumber]
		if !found {
			m.log.Warnf("No instance found for phone %s, skipping device", phoneNumber)
			continue
		}

		// Create client with the correct instance name
		client, err := m.createClientWithDevice(instanceInfo.Name, device)
		if err != nil {
			m.log.Errorf("Failed to restore client for instance %s: %v", instanceInfo.Name, err)
			continue
		}

		// Configure settings
		if instanceInfo.WebhookURL != "" {
			client.SetWebhook(instanceInfo.WebhookURL)
		}
		if instanceInfo.ProxyURL != "" {
			if err := client.SetProxy(instanceInfo.ProxyURL); err != nil {
				m.log.Errorf("Failed to set proxy for instance %s: %v", instanceInfo.Name, err)
			}
		}
		client.SetIgnoreGroups(instanceInfo.IgnoreGroups)

		m.log.Infof("Restored client for instance %s (phone: %s)", instanceInfo.Name, phoneNumber)
		count++

		// Return the instance name so caller can reconnect
		_ = client
	}

	m.log.Infof("Restored %d clients from database", count)
}

// createClientWithDevice creates a client using an existing device store
func (m *InstanceManager) createClientWithDevice(instanceID string, deviceStore *store.Device) (*WAClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	if client, exists := m.clients[instanceID]; exists {
		return client, nil
	}

	// Create whatsmeow client with the existing device
	shortID := instanceID
	if len(instanceID) > 8 {
		shortID = instanceID[:8]
	}
	clientLog := waLog.Stdout("Client-"+shortID, "INFO", true)
	waClient := whatsmeow.NewClient(deviceStore, clientLog)

	// Create our wrapper
	client := NewWAClient(instanceID, waClient, m.db)
	m.clients[instanceID] = client

	return client, nil
}

// GetClient returns an existing client or creates a new one for the given instance ID
func (m *InstanceManager) GetClient(instanceID string) (*WAClient, error) {
	m.mu.RLock()
	if client, exists := m.clients[instanceID]; exists {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	// If not found, create a new client without a specific phone number
	// This will create a new device store if one doesn't exist for this instanceID
	return m.createClient(instanceID, "")
}

// createClient creates a new WhatsApp client for the given instance ID
func (m *InstanceManager) createClient(instanceID string, phoneNumber string) (*WAClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := m.clients[instanceID]; exists {
		return client, nil
	}

	// Get or create device store for this instance
	deviceStore, err := m.getOrCreateDeviceStore(instanceID, phoneNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	// Create whatsmeow client
	shortID := instanceID
	if len(instanceID) > 8 {
		shortID = instanceID[:8]
	}
	clientLog := waLog.Stdout("Client-"+shortID, "INFO", true)
	waClient := whatsmeow.NewClient(deviceStore, clientLog)

	// Create our wrapper
	client := NewWAClient(instanceID, waClient, m.db)

	m.clients[instanceID] = client

	return client, nil
}

// getOrCreateDeviceStore gets an existing device store or creates a new one
func (m *InstanceManager) getOrCreateDeviceStore(instanceID string, phoneNumber string) (*store.Device, error) {
	// If we have a phone number, try to find the existing device by JID
	if phoneNumber != "" {
		jid := types.NewJID(phoneNumber, types.DefaultUserServer)
		device, err := m.container.GetDevice(m.ctx, jid)
		if err != nil {
			return nil, fmt.Errorf("failed to get device by JID %s: %w", jid.String(), err)
		}
		if device != nil {
			// Found an existing device for this phone number
			return device, nil
		}
	}

	// If no phone number was provided, or no device was found by phone number,
	// try to find a device associated with the instanceID.
	// This assumes instanceID is stored somewhere, e.g., in the device's ID or a custom field.
	// For now, we'll just create a new device if not found by phone number.
	// A more robust solution might involve a custom lookup table for instanceID -> JID.

	// Create new device store
	deviceStore := m.container.NewDevice()
	// Optionally, you could store the instanceID in a custom field if the store supports it,
	// or rely on the client's JID once connected to identify it.
	// For now, we're relying on the client map (m.clients) to link instanceID to a WAClient,
	// and the WAClient holds the whatsmeow.Client which has the device store.
	return deviceStore, nil
}

// Connect initiates the connection process for an instance
func (m *InstanceManager) Connect(ctx context.Context, instanceID string) (*WAClient, error) {
	client, err := m.GetClient(instanceID)
	if err != nil {
		return nil, err
	}

	// Start connection (this will trigger QR code generation if not logged in)
	// Use manager context to ensure connection persists after request
	err = client.Connect(m.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return client, nil
}

// Disconnect disconnects an instance and removes it from memory
func (m *InstanceManager) Disconnect(instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[instanceID]

	if !exists {
		// If client doesn't exist in memory, consider it already disconnected
		return nil
	}

	client.Disconnect()

	// Remove client from memory to ensure clean state on reconnection
	delete(m.clients, instanceID)
	return nil
}

// RemoveClient removes a client from the manager
// RemoveClient removes a client from the manager and deletes its data
func (m *InstanceManager) RemoveClient(instanceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// First, handle in-memory client if it exists
	if client, exists := m.clients[instanceID]; exists {
		// Disconnect first
		client.Disconnect()

		// Delete device data from database via the client's store
		if client.client != nil && client.client.Store != nil {
			err := client.client.Store.Delete(m.ctx)
			if err != nil {
				m.log.Errorf("Failed to delete device data for instance %s: %v", instanceID, err)
			} else {
				m.log.Infof("Deleted device data for instance %s", instanceID)
			}
		}

		delete(m.clients, instanceID)
	}

}

// GetQRCode returns the current QR code for an instance (base64 encoded)
func (m *InstanceManager) GetQRCode(instanceID string) (string, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.GetQRCode(), nil
}

// GetSyncProgress returns the history sync progress (0-100) for an instance
func (m *InstanceManager) GetSyncProgress(instanceID string) int {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()
	if !exists {
		return 0
	}
	return client.GetSyncProgress()
}

// HasClient returns whether an instance client is loaded in memory
func (m *InstanceManager) HasClient(instanceID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.clients[instanceID]
	return exists
}

// GetStatus returns the connection status for an instance
func (m *InstanceManager) GetStatus(instanceID string) (string, string, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return "disconnected", "", nil
	}

	status, phone := client.GetStatus()
	return status, phone, nil
}

// Close closes all clients and the container
func (m *InstanceManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		client.Disconnect()
	}

	m.clients = make(map[string]*WAClient)
}

// SendTextMessage sends a text message through the specified instance
func (m *InstanceManager) SendTextMessage(instanceID string, recipient string, text string, simulateTyping bool, typingDurationMs int) (*SendResponse, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.SendTextMessage(m.ctx, recipient, text, simulateTyping, typingDurationMs)
}

// SendImageMessage sends an image through the specified instance
func (m *InstanceManager) SendImageMessage(instanceID string, recipient string, imageData []byte, mimeType string, caption string) (*SendResponse, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.SendImageMessage(m.ctx, recipient, imageData, mimeType, caption)
}

// SendVideoMessage sends a video through the specified instance
func (m *InstanceManager) SendVideoMessage(instanceID string, recipient string, videoData []byte, mimeType string, caption string) (*SendResponse, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.SendVideoMessage(m.ctx, recipient, videoData, mimeType, caption)
}

// SendAudioMessage sends an audio file through the specified instance
func (m *InstanceManager) SendAudioMessage(instanceID string, recipient string, audioData []byte, mimeType string, simulateRecording bool, recordingDurationMs int, ptt bool) (*SendResponse, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.SendAudioMessage(m.ctx, recipient, audioData, mimeType, simulateRecording, recordingDurationMs, ptt)
}

// SendDocumentMessage sends a document through the specified instance
func (m *InstanceManager) SendDocumentMessage(instanceID string, recipient string, docData []byte, mimeType string, fileName string, caption string) (*SendResponse, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.SendDocumentMessage(m.ctx, recipient, docData, mimeType, fileName, caption)
}

// ConnectWithPhone creates (or reuses) a client for instanceID using a known phone number
// to find the existing whatsmeow device store, then initiates connection.
// Used for auto-reconnect when the client is not in memory but a saved session exists.
// Uses GetAllDevices + ID.User matching (like RestoreClients) to find the AD-format JID device.
func (m *InstanceManager) ConnectWithPhone(instanceID, phoneNumber string) (*WAClient, error) {
	m.mu.RLock()
	if client, exists := m.clients[instanceID]; exists {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	// Scan all devices to find the one matching the phone number.
	// Devices are stored with AD JIDs (e.g. "5511999:12@s.whatsapp.net"), so we can't
	// query by a simple non-AD JID — we must scan and match by device.ID.User.
	devices, err := m.container.GetAllDevices(m.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	var foundDevice *store.Device
	for _, device := range devices {
		if device.ID != nil && device.ID.User == phoneNumber {
			foundDevice = device
			break
		}
	}
	if foundDevice == nil {
		return nil, fmt.Errorf("no saved session found for phone %s", phoneNumber)
	}

	client, err := m.createClientWithDevice(instanceID, foundDevice)
	if err != nil {
		return nil, err
	}

	// Load settings from DB (webhook URL, ignore groups, proxy)
	if m.db != nil {
		var inst struct {
			WebhookUrl   *string
			IgnoreGroups *bool
			ProxyEnabled *bool
			ProxyUrl     *string
		}
		if err := m.db.Table("instances").
			Select("webhook_url, ignore_groups, proxy_enabled, proxy_url").
			Where("name = ?", instanceID).
			Scan(&inst).Error; err == nil {
			if inst.WebhookUrl != nil && *inst.WebhookUrl != "" {
				client.SetWebhook(*inst.WebhookUrl)
			}
			if inst.IgnoreGroups != nil {
				client.SetIgnoreGroups(*inst.IgnoreGroups)
			}
			if inst.ProxyEnabled != nil && *inst.ProxyEnabled && inst.ProxyUrl != nil {
				_ = client.SetProxy(*inst.ProxyUrl)
			}
		}
	}

	if err := client.Connect(m.ctx); err != nil {
		return nil, err
	}
	return client, nil
}

// GetContacts retrieves contacts for the specified instance
func (m *InstanceManager) GetContacts(instanceID string) ([]ContactInfo, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.GetContacts(m.ctx)
}

// GetProfilePictureURL returns the profile picture URL for the given phone number.
func (m *InstanceManager) GetProfilePictureURL(instanceID string, phone string) (string, error) {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.GetProfilePictureURL(m.ctx, phone)
}

// SetPresence sets the online/offline presence for the specified instance
func (m *InstanceManager) SetPresence(instanceID string, available bool) error {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found for instance %s", instanceID)
	}

	return client.SetPresence(available)
}
