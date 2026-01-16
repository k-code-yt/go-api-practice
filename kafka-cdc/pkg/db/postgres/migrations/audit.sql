CREATE TABLE event_inbox (
    id SERIAL PRIMARY KEY,
    type VARCHAR NOT NULL,
    aggregate_id VARCHAR NOT NULL,
    aggregate_created_at TIMESTAMP NOT NULL,
    aggregate_type VARCHAR NOT NULL,
    aggregate_metadata JSONB,
    created_at TIMESTAMP NOT NULL,
    status VARCHAR NOT NULL
);

