package whatsapp

import (
	"context"
	"fmt"
	"sync"

	_ "github.com/lib/pq" // Register PostgreSQL driver
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
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

	return &InstanceManager{
		container: container,
		clients:   make(map[string]*WAClient),
		log:       log,
		ctx:       ctx,
	}, nil
}

// GetClient returns an existing client or creates a new one for the given instance ID
func (m *InstanceManager) GetClient(instanceID string) (*WAClient, error) {
	m.mu.RLock()
	if client, exists := m.clients[instanceID]; exists {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	return m.createClient(instanceID)
}

// createClient creates a new WhatsApp client for the given instance ID
func (m *InstanceManager) createClient(instanceID string) (*WAClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := m.clients[instanceID]; exists {
		return client, nil
	}

	// Get or create device store for this instance
	deviceStore, err := m.getOrCreateDeviceStore(instanceID)
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
func (m *InstanceManager) getOrCreateDeviceStore(instanceID string) (*store.Device, error) {
	// Try to get existing device by instance ID
	devices, err := m.container.GetAllDevices(m.ctx)
	if err != nil {
		return nil, err
	}

	// Look for device with matching instance ID in the devices
	for _, device := range devices {
		// We store instance ID in the device's PushName or use JID as identifier
		if device.ID != nil && device.ID.String() != "" {
			// For now, we'll create new devices for each instance
			// In production, you'd want to map instance IDs to device JIDs
		}
	}

	// Create new device store
	deviceStore := m.container.NewDevice()
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

// Disconnect disconnects an instance
func (m *InstanceManager) Disconnect(instanceID string) error {
	m.mu.RLock()
	client, exists := m.clients[instanceID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found for instance %s", instanceID)
	}

	client.Disconnect()
	return nil
}

// RemoveClient removes a client from the manager
func (m *InstanceManager) RemoveClient(instanceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.clients[instanceID]; exists {
		client.Disconnect()
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
