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
)

// InstanceManager manages multiple WhatsApp client instances
type InstanceManager struct {
	container *sqlstore.Container
	clients   map[string]*WAClient
	mu        sync.RWMutex
	log       waLog.Logger
	ctx       context.Context
}

// NewInstanceManager creates a new WhatsApp instance manager
func NewInstanceManager(dbURL string) (*InstanceManager, error) {
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

	log.Infof("WhatsApp store initialized successfully")

	manager := &InstanceManager{
		container: container,
		clients:   make(map[string]*WAClient),
		log:       log,
		ctx:       ctx,
	}

	// Restore clients from database
	manager.restoreClients()

	return manager, nil
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
	client := NewWAClient(instanceID, waClient)

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

// restoreClients loads existing clients from the database
func (m *InstanceManager) restoreClients() {
	// In the new model, we don't rely on PushName.
	// We need a way to map stored devices back to instanceIDs.
	// A simple approach is to iterate all devices and create a client for each.
	// If instanceID is not explicitly stored with the device, we might use the JID or a generated ID.
	// For this change, we'll assume instanceID is implicitly tied to the device's JID
	// or that we're restoring based on existing device records.
	// If a device has a JID, we can use that to identify it.

	devices, err := m.container.GetAllDevices(m.ctx)
	if err != nil {
		m.log.Errorf("Failed to get devices for restoration: %v", err)
		return
	}

	count := 0
	for _, device := range devices {
		// If the device has an ID (meaning it's been logged in before),
		// we can use its JID to identify it.
		// We'll use the JID as the instanceID for restoration purposes.
		if device.ID != nil {
			instanceID := device.ID.String() // Use JID as instanceID for restored clients

			// Create client (this populates m.clients)
			// Pass the JID's user part as phoneNumber for lookup, if available
			phoneNumber := device.ID.User
			client, err := m.createClient(instanceID, phoneNumber)
			if err != nil {
				m.log.Errorf("Failed to restore client for instance %s (JID: %s): %v", instanceID, device.ID.String(), err)
				continue
			}

			// If client has session, connect it
			if client.client.Store.ID != nil {
				go func(id string) {
					if _, err := m.Connect(m.ctx, id); err != nil {
						m.log.Errorf("Failed to auto-connect instance %s: %v", id, err)
					}
				}(instanceID)
			}
			count++
		}
	}
	m.log.Infof("Restored %d clients from database", count)
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

// Disconnect disconnects an instance
func (m *InstanceManager) Disconnect(instanceID string) error {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		// If client doesn't exist in memory, consider it already disconnected
		return nil
	}

	client.Disconnect()
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

	// Additionally, clean up ALL orphaned devices from the database
	// This handles cases where the client was never in memory (e.g., after restart)
	devices, err := m.container.GetAllDevices(m.ctx)
	if err != nil {
		m.log.Errorf("Failed to get devices for cleanup: %v", err)
		return
	}

	for _, device := range devices {
		err := device.Delete(m.ctx)
		if err != nil {
			m.log.Errorf("Failed to delete orphaned device %v: %v", device.ID, err)
		} else {
			m.log.Infof("Deleted orphaned device %v", device.ID)
		}
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
