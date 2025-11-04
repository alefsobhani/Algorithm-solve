-- name: CreateTrip :one
INSERT INTO trips (
    id, rider_id, driver_id, pickup, dropoff, status,
    requested_at, accepted_at, started_at, finished_at,
    cancelled_at, price_cents, version
) VALUES (
    $1, $2, $3, ST_SetSRID(ST_Point($4, $5), 4326), ST_SetSRID(ST_Point($6, $7), 4326), $8,
    $9, $10, $11, $12, $13, $14, $15
)
RETURNING *;

-- name: GetTrip :one
SELECT * FROM trips WHERE id = $1 LIMIT 1;

-- name: UpdateTrip :one
UPDATE trips
SET
    rider_id = $2,
    driver_id = $3,
    pickup = ST_SetSRID(ST_Point($4, $5), 4326),
    dropoff = ST_SetSRID(ST_Point($6, $7), 4326),
    status = $8,
    requested_at = $9,
    accepted_at = $10,
    started_at = $11,
    finished_at = $12,
    cancelled_at = $13,
    price_cents = $14,
    version = version + 1
WHERE id = $1
RETURNING *;
