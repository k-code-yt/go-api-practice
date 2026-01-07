CREATE TABLE audit (
    id SERIAL PRIMARY KEY,
    type VARCHAR NOT NULL,
    aggregate_id VARCHAR NOT NULL,
    aggregate_created_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    status VARCHAR NOT NULL,
    parent_id VARCHAR NOT NULL,
    parent_type VARCHAR NOT NULL,
    parent_metadata JSONB
);

