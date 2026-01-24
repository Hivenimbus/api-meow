-- Script para resetar completamente as tabelas do whatsmeow
-- ATENÇÃO: Isso irá deletar todas as sessões do WhatsApp!
-- Você precisará reconectar todas as instâncias via QR Code.

-- Dropar tabelas existentes do whatsmeow (ordem importante por causa das foreign keys)
DROP TABLE IF EXISTS whatsmeow_chat_settings CASCADE;
DROP TABLE IF EXISTS whatsmeow_contacts CASCADE;
DROP TABLE IF EXISTS whatsmeow_app_state_version CASCADE;
DROP TABLE IF EXISTS whatsmeow_app_state_sync_keys CASCADE;
DROP TABLE IF EXISTS whatsmeow_sessions CASCADE;
DROP TABLE IF EXISTS whatsmeow_sender_keys CASCADE;
DROP TABLE IF EXISTS whatsmeow_pre_keys CASCADE;
DROP TABLE IF EXISTS whatsmeow_identity_keys CASCADE;
DROP TABLE IF EXISTS whatsmeow_device CASCADE;
DROP TABLE IF EXISTS whatsmeow_lid_lookup CASCADE;

-- Dropar qualquer outra tabela que o whatsmeow possa ter criado
DROP TABLE IF EXISTS whatsmeow_version CASCADE;

-- Limpar instâncias da sua aplicação (opcional - remova se quiser manter os registros)
-- DELETE FROM instances;

-- Após executar este script:
-- 1. Reinicie o backend (go run . ou npm run dev)
-- 2. O whatsmeow irá recriar todas as tabelas com o schema correto
-- 3. Reconecte suas instâncias via /connect endpoint
