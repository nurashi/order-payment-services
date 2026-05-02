-- +goose Up
CREATE TABLE IF NOT EXISTS processed_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_processed_events_event_id ON processed_events(event_id);

-- +goose Down
DROP INDEX IF EXISTS idx_processed_events_event_id;
DROP TABLE IF EXISTS processed_events;
