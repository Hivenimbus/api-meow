-- Unified Database Schema (excluding whatsmeow tables)
-- This schema consolidates all application tables into a single file

-- ============================================
-- TABLE: instances
-- Manages WhatsApp instances/connections
-- ============================================
CREATE TABLE instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'disconnected',
    phone_number TEXT,
    tag_id TEXT,
    ignore_groups BOOLEAN DEFAULT TRUE,
    webhook_url TEXT,
    receive_messages BOOLEAN DEFAULT TRUE,
    proxy_enabled BOOLEAN DEFAULT FALSE,
    proxy_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT instances_name_key UNIQUE (name)
);

-- ============================================
-- TABLE: tags
-- Manages tags/labels for categorization
-- ============================================
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
