-- Mostly vibecoded
BEGIN;

CREATE SCHEMA IF NOT EXISTS order_service;

-- Enums for fun, wrap in a nested transaction to implement "if not exists"
DO
$$
    BEGIN
        CREATE TYPE order_status AS ENUM ('pending', 'processing', 'completed', 'cancelled');
    EXCEPTION
        WHEN duplicate_object THEN null;
    END
$$;
-- No locale/currency enum because there're too much of them

CREATE TABLE IF NOT EXISTS order_service.orders
(
    order_uid          VARCHAR(50)              NOT NULL PRIMARY KEY,
    track_number       VARCHAR(50)              NOT NULL,
    entry              VARCHAR(10)              NOT NULL,
    locale             VARCHAR(2)               NOT NULL,
    internal_signature VARCHAR(100)             NOT NULL,
    customer_id        VARCHAR(50)              NOT NULL,
    delivery_service   VARCHAR(50)              NOT NULL,
    shardkey           VARCHAR(10)              NOT NULL,
    sm_id              INTEGER                  NOT NULL,
    date_created       TIMESTAMP WITH TIME ZONE NOT NULL,
    oof_shard          VARCHAR(10)              NOT NULL,
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, -- good practice as I've been told
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP  -- good practice as I've been told
);

CREATE TABLE IF NOT EXISTS order_service.deliveries
(
    order_id VARCHAR(50) PRIMARY KEY REFERENCES orders (order_uid) ON DELETE CASCADE,
    name     VARCHAR(100) NOT NULL,
    phone    VARCHAR(20)  NOT NULL,
    zip      VARCHAR(20)  NOT NULL,
    city     VARCHAR(100) NOT NULL,
    address  VARCHAR(200) NOT NULL,
    region   VARCHAR(100) NOT NULL,
    email    VARCHAR(100) NOT NULL
    -- no created/updated time because who would change this?
    -- my granny who accidentally granted the access?
);

CREATE TABLE IF NOT EXISTS order_service.payments
(
    order_id      VARCHAR(50) PRIMARY KEY REFERENCES orders (order_uid) ON DELETE CASCADE,
    transaction   VARCHAR(50) NOT NULL,
    request_id    VARCHAR(50) NOT NULL,
    currency      VARCHAR(3)  NOT NULL,
    provider      VARCHAR(50) NOT NULL,
    amount        INTEGER     NOT NULL CHECK (amount >= 0),
    payment_dt    BIGINT      NOT NULL,
    bank          VARCHAR(50) NOT NULL,
    delivery_cost INTEGER     NOT NULL CHECK (delivery_cost >= 0),
    goods_total   INTEGER     NOT NULL CHECK (goods_total >= 0),
    custom_fee    INTEGER     NOT NULL CHECK (custom_fee >= 0)
);

CREATE TABLE IF NOT EXISTS order_service.order_items
(
    order_id     VARCHAR(50)  NOT NULL REFERENCES orders (order_uid) ON DELETE CASCADE,
    chrt_id      BIGINT       NOT NULL,
    track_number VARCHAR(50)  NOT NULL,
    price        INTEGER      NOT NULL CHECK (price >= 0),
    rid          VARCHAR(50)  NOT NULL,
    name         VARCHAR(100) NOT NULL,
    sale         INTEGER      NOT NULL CHECK (sale >= 0 AND sale <= 100),
    size         VARCHAR(10)  NOT NULL,
    total_price  INTEGER      NOT NULL CHECK (total_price >= 0),
    nm_id        BIGINT       NOT NULL,
    brand        VARCHAR(100) NOT NULL,
    status       INTEGER      NOT NULL,

    PRIMARY KEY (order_id, nm_id)
);

-- Create indexes for the queried field: order ID (and the foreign key)
CREATE INDEX IF NOT EXISTS idx_orders_order_uid ON order_service.orders (order_uid);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_service.order_items (order_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_order_id ON order_service.deliveries (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_order_id ON order_service.payments (order_id);

-- Function to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to call the function that updates updated_at
CREATE OR REPLACE TRIGGER update_orders_updated_at
    BEFORE UPDATE
    ON order_service.orders
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

COMMIT;