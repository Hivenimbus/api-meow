-- +goose Up
CREATE TABLE IF NOT EXISTS whatsmeow_device (
    jid text NOT NULL,
    registration_id integer NOT NULL,
    noise_key bytea NOT NULL,
    identity_key bytea NOT NULL,
    signed_pre_key_id integer NOT NULL,
    signed_pre_key bytea NOT NULL,
    signed_pre_key_sig bytea NOT NULL,
    adv_key_secret bytea NOT NULL,
    adv_key_index integer NOT NULL,
    adv_account_sig bytea NOT NULL,
    platform text NOT NULL,
    business_name text NOT NULL,
    push_name text NOT NULL,
    ad_account_id text NOT NULL,
    verified_name bytea,
    verified_name_cert bytea,
    verified_name_ver bytea,
    pair_proto_version integer NOT NULL,
    PRIMARY KEY (jid)
);

CREATE TABLE IF NOT EXISTS whatsmeow_identity_keys (
    our_jid text NOT NULL,
    their_jid text NOT NULL,
    content bytea NOT NULL,
    PRIMARY KEY (our_jid, their_jid)
);

CREATE TABLE IF NOT EXISTS whatsmeow_pre_keys (
    our_jid text NOT NULL,
    key_id integer NOT NULL,
    content bytea NOT NULL,
    upload_time bigint NOT NULL,
    uploaded boolean NOT NULL,
    PRIMARY KEY (our_jid, key_id)
);

CREATE TABLE IF NOT EXISTS whatsmeow_sender_keys (
    our_jid text NOT NULL,
    group_id text NOT NULL,
    sender_id text NOT NULL,
    content bytea NOT NULL,
    PRIMARY KEY (our_jid, group_id, sender_id)
);

CREATE TABLE IF NOT EXISTS whatsmeow_sessions (
    our_jid text NOT NULL,
    their_jid text NOT NULL,
    content bytea NOT NULL,
    PRIMARY KEY (our_jid, their_jid)
);

CREATE TABLE IF NOT EXISTS whatsmeow_app_state_sync_keys (
    our_jid text NOT NULL,
    key_id bytea NOT NULL,
    key_data bytea NOT NULL,
    timestamp bigint NOT NULL,
    fingerprint bytea NOT NULL,
    PRIMARY KEY (our_jid, key_id)
);

CREATE TABLE IF NOT EXISTS whatsmeow_app_state_version (
    our_jid text NOT NULL,
    name text NOT NULL,
    version bigint NOT NULL,
    hash bytea NOT NULL,
    PRIMARY KEY (our_jid, name)
);

CREATE TABLE IF NOT EXISTS whatsmeow_contacts (
    our_jid text NOT NULL,
    their_jid text NOT NULL,
    first_name text NOT NULL,
    full_name text NOT NULL,
    push_name text NOT NULL,
    business_name text NOT NULL,
    row_id bigint NOT NULL,
    PRIMARY KEY (our_jid, their_jid)
);

CREATE TABLE IF NOT EXISTS whatsmeow_chat_settings (
    our_jid text NOT NULL,
    chat_jid text NOT NULL,
    muted_until bigint NOT NULL,
    pinned boolean NOT NULL,
    archived boolean NOT NULL,
    PRIMARY KEY (our_jid, chat_jid)
);

-- +goose Down
DROP TABLE IF EXISTS whatsmeow_chat_settings;
DROP TABLE IF EXISTS whatsmeow_contacts;
DROP TABLE IF EXISTS whatsmeow_app_state_version;
DROP TABLE IF EXISTS whatsmeow_app_state_sync_keys;
DROP TABLE IF EXISTS whatsmeow_sessions;
DROP TABLE IF EXISTS whatsmeow_sender_keys;
DROP TABLE IF EXISTS whatsmeow_pre_keys;
DROP TABLE IF EXISTS whatsmeow_identity_keys;
DROP TABLE IF EXISTS whatsmeow_device;
