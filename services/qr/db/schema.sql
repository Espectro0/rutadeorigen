CREATE TABLE IF NOT EXISTS lot_view (
    lot_id          UUID        PRIMARY KEY,
    code            TEXT        NOT NULL,
    variety         TEXT        NOT NULL,
    process         TEXT        NOT NULL CHECK (process IN ('washed', 'natural', 'honey')),
    harvest_date    DATE        NOT NULL,
    farm_id         UUID        NOT NULL,
    farm_name       TEXT        NOT NULL,
    farm_altitude   INTEGER     NOT NULL CHECK (farm_altitude BETWEEN 0 AND 3000),
    producer_id     UUID        NOT NULL,
    producer_name   TEXT        NOT NULL,
    custodian_id    UUID        NOT NULL,
    current_stage   TEXT        NOT NULL CHECK (current_stage IN ('none', 'farm', 'roasting', 'packaging')),
    status          TEXT        NOT NULL CHECK (status IN ('active', 'voided')),
    void_reason     TEXT        NOT NULL DEFAULT '',
    version         INTEGER     NOT NULL CHECK (version > 0),
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS qr_code (
    lot_id            UUID        PRIMARY KEY REFERENCES lot_view (lot_id),
    content           TEXT        NOT NULL UNIQUE,
    error_correction  TEXT        NOT NULL CHECK (error_correction IN ('L', 'M', 'Q', 'H')),
    library_version   TEXT        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS redirect (
    id               UUID        PRIMARY KEY,
    source_prefix    TEXT        NOT NULL UNIQUE,
    target_template  TEXT        NOT NULL,
    status_code      INTEGER     NOT NULL CHECK (status_code IN (301, 302, 308)),
    active           BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS qr_scan (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lot_id      UUID        NOT NULL REFERENCES lot_view (lot_id),
    scanned_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS qr_scan_lot_scanned_at_idx ON qr_scan (lot_id, scanned_at DESC);

CREATE TABLE IF NOT EXISTS processed_event (
    message_id    UUID        PRIMARY KEY,
    event_type    TEXT        NOT NULL,
    processed_at  TIMESTAMPTZ NOT NULL
);

INSERT INTO redirect (id, source_prefix, target_template, status_code, active, created_at)
VALUES ('0198a000-0000-7000-8000-000000000001', '/l/', '/lotes/{id}', 301, TRUE, NOW())
ON CONFLICT (source_prefix) DO NOTHING;
