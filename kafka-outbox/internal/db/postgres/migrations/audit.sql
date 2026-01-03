CREATE TABLE audit (
    id SERIAL PRIMARY KEY,
    type VARCHAR NOT NULL,
    outbox_event_id VARCHAR NOT NULL,
    outbox_created_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    status VARCHAR NOT NULL,
    parent_id VARCHAR NOT NULL,
    parent_type VARCHAR NOT NULL,
    parent_metadata JSONB
);

ALTER TABLE audit
ALTER COLUMN outbox_event_id SET NOT NULL,
ADD CONSTRAINT audit_outbox_event_id_unique UNIQUE (outbox_event_id);