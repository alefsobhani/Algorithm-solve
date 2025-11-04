-- +migrate Up
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE users (
    id UUID PRIMARY KEY,
    role TEXT CHECK (role IN ('rider','driver','admin')) NOT NULL,
    phone TEXT UNIQUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE drivers (
    id UUID PRIMARY KEY REFERENCES users(id),
    vehicle_type TEXT,
    status TEXT,
    updated_at TIMESTAMPTZ
);

CREATE TABLE trips (
    id UUID PRIMARY KEY,
    rider_id UUID REFERENCES users(id),
    driver_id UUID REFERENCES users(id),
    pickup GEOMETRY(POINT, 4326),
    dropoff GEOMETRY(POINT, 4326),
    status TEXT,
    requested_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    price_cents INT,
    version INT DEFAULT 1
);

CREATE TABLE trip_events (
    id BIGSERIAL PRIMARY KEY,
    trip_id UUID REFERENCES trips(id),
    event_type TEXT,
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE idempotency_keys (
    key TEXT PRIMARY KEY,
    request_hash TEXT,
    response JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    topic TEXT,
    payload JSONB,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- +migrate Down
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS trip_events;
DROP TABLE IF EXISTS trips;
DROP TABLE IF EXISTS drivers;
DROP TABLE IF EXISTS users;
