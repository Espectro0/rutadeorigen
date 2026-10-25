ALTER TABLE lot_event ADD COLUMN IF NOT EXISTS details JSONB NOT NULL DEFAULT '{}'

CREATE TABLE IF NOT EXISTS actor (
                                     id          UUID        PRIMARY KEY,
                                     user_id     TEXT        NOT NULL UNIQUE,
                                     type        TEXT        NOT NULL CHECK (type IN ('organization', 'producer', 'roaster', 'brand')),
    name        TEXT        NOT NULL,
    status      TEXT        NOT NULL CHECK (status IN ('active', 'inactive')),
    created_at  TIMESTAMPTZ NOT NULL
    );

CREATE TABLE IF NOT EXISTS farm (
                                    id           UUID        PRIMARY KEY,
                                    producer_id  UUID        NOT NULL REFERENCES actor (id),
    name         TEXT        NOT NULL,
    altitude     INTEGER     NOT NULL CHECK (altitude BETWEEN 0 AND 3000),
    created_at   TIMESTAMPTZ NOT NULL
    );

CREATE INDEX IF NOT EXISTS farm_producer_id_idx ON farm (producer_id);

CREATE TABLE IF NOT EXISTS lot (
                                   id             UUID        PRIMARY KEY,
                                   farm_id        UUID        NOT NULL REFERENCES farm (id),
    custodian_id   UUID        NOT NULL REFERENCES actor (id),
    code           TEXT        NOT NULL,
    variety        TEXT        NOT NULL,
    process        TEXT        NOT NULL CHECK (process IN ('washed', 'natural', 'honey')),
    harvest_date   DATE        NOT NULL,
    current_stage  TEXT        NOT NULL CHECK (current_stage IN ('none', 'farm', 'roasting', 'packaging')),
    status         TEXT        NOT NULL CHECK (status IN ('active', 'voided')),
    void_reason    TEXT        NOT NULL DEFAULT '',
    version        INTEGER     NOT NULL CHECK (version > 0),
    created_at     TIMESTAMPTZ NOT NULL,
    CONSTRAINT lot_farm_code_uk UNIQUE (farm_id, code)
    );

DROP INDEX IF EXISTS lot_custodian_id_idx;
CREATE INDEX IF NOT EXISTS lot_created_at_id_idx           ON lot (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS lot_farm_created_at_id_idx      ON lot (farm_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS lot_custodian_created_at_id_idx ON lot (custodian_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS lot_event (
                                         id                 UUID        PRIMARY KEY,
                                         lot_id             UUID        NOT NULL REFERENCES lot (id),
    sequence           INTEGER     NOT NULL CHECK (sequence > 0),
    type               TEXT        NOT NULL CHECK (type IN ('stage', 'transfer', 'correction')),
    stage              TEXT        NOT NULL DEFAULT '',
    event_date         DATE        NOT NULL,
    note               TEXT        NOT NULL DEFAULT '',
    actor_id           UUID        NOT NULL REFERENCES actor (id),
    actor_role         TEXT        NOT NULL,
    target_actor_id    UUID        REFERENCES actor (id),
    corrects_event_id  UUID        REFERENCES lot_event (id),
    previous_hash      TEXT        NOT NULL,
    hash               TEXT        NOT NULL UNIQUE,
    registered_at      TIMESTAMPTZ NOT NULL,
    CONSTRAINT lot_event_lot_sequence_uk UNIQUE (lot_id, sequence)
    );

CREATE TABLE IF NOT EXISTS outbox (
                                      id              UUID        PRIMARY KEY,
                                      aggregate_type  TEXT        NOT NULL,
                                      aggregate_id    UUID        NOT NULL,
                                      event_type      TEXT        NOT NULL,
                                      version         INTEGER     NOT NULL,
                                      payload         JSONB       NOT NULL,
                                      created_at      TIMESTAMPTZ NOT NULL,
                                      published_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS outbox_unpublished_idx ON outbox (created_at) WHERE published_at IS NULL;