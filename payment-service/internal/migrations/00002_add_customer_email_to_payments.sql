-- +goose Up
ALTER TABLE payments ADD COLUMN IF NOT EXISTS customer_email VARCHAR(255);

-- +goose Down
ALTER TABLE payments DROP COLUMN IF EXISTS customer_email;
