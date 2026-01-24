-- +goose Up
-- Adds the LID lookup table required by newer versions of whatsmeow
CREATE TABLE IF NOT EXISTS whatsmeow_lid_lookup (
    our_jid text NOT NULL,
    their_jid text NOT NULL,
    lid text NOT NULL,
    server text NOT NULL,
    PRIMARY KEY (our_jid, their_jid, lid, server)
);

-- +goose Down
DROP TABLE IF EXISTS whatsmeow_lid_lookup;
