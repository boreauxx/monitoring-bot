-- +goose Up

-- Tables
CREATE TABLE IF NOT EXISTS assets
(
    id               UUID PRIMARY KEY     DEFAULT GEN_RANDOM_UUID(),
    name             TEXT        NOT NULL,
    address          TEXT        NOT NULL UNIQUE,
    interval_seconds INTEGER     NOT NULL DEFAULT 30,
    timeout_seconds  INTEGER     NOT NULL DEFAULT 5,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS probe_events
(
    id          UUID PRIMARY KEY     DEFAULT GEN_RANDOM_UUID(),
    success     BOOLEAN     NOT NULL,
    code        INTEGER     NOT NULL,
    err_message TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    asset_id    UUID        NOT NULL REFERENCES assets (id)
);

CREATE TABLE IF NOT EXISTS incidents
(
    id         UUID PRIMARY KEY     DEFAULT GEN_RANDOM_UUID(),
    severity   TEXT                 DEFAULT 'INITIAL',
    summary    TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at   TIMESTAMPTZ,
    asset_id   UUID        NOT NULL REFERENCES assets (id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_probe_events_asset_id_created_at ON probe_events (asset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_asset_id ON incidents (asset_id);
CREATE INDEX IF NOT EXISTS idx_incidents_ended_at ON incidents (ended_at);
CREATE UNIQUE INDEX IF NOT EXISTS uq_incidents_open_asset ON incidents (asset_id) WHERE ended_at IS NULL;

-- +goose Down

DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS probe_events;
DROP TABLE IF EXISTS assets;
