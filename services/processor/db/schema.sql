CREATE TABLE IF NOT EXISTS processed_event (
                                               message_id    UUID        PRIMARY KEY,
                                               event_type    TEXT        NOT NULL,
                                               processed_at  TIMESTAMPTZ NOT NULL
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
