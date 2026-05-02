-- +goose Up
ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_email VARCHAR(255);

-- +goose Down
ALTER TABLE orders DROP COLUMN IF EXISTS customer_email;
