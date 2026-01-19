CREATE TABLE payment_orders (
    id SERIAL PRIMARY KEY,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE events 
ADD COLUMN status VARCHAR(50),
ADD COLUMN parent_id INTEGER,
ADD COLUMN parent_type VARCHAR(100),
ADD COLUMN parent_metadata JSONB;

ALTER TABLE events 
DROP COLUMN parent_id;

ALTER TABLE events 
ADD COLUMN parent_id INTEGER;


select * from payment_orders

select "event_id", status from events

LEFT JOIN payment_orders po
ON events.parent_id = po.id

SELECT current_database();

