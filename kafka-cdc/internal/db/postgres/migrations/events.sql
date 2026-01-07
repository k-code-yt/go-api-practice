CREATE TABLE events (
    event_id VARCHAR PRIMARY KEY,
    event_type VARCHAR NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR NOT NULL,
    parent_id VARCHAR,
    parent_type VARCHAR,
    parent_metadata JSONB
);