CREATE TABLE inventory (
	id SERIAL PRIMARY KEY,
    product_name VARCHAR(255),
    quantity INT,
	status VARCHAR(20) DEFAULT 'reserved',
    last_updated DATE,

	payment_id VARCHAR,
	order_number VARCHAR(50) UNIQUE NOT NULL -- payment's order_number
);