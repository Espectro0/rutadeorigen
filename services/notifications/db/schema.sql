CREATE TABLE IF NOT EXISTS processed_event (
    message_id    UUID        PRIMARY KEY,
    event_type    TEXT        NOT NULL,
    processed_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_log (
    id               UUID        PRIMARY KEY,
    evidence_id      UUID        NOT NULL,
    kind             TEXT        NOT NULL,
    recipient_name   TEXT        NOT NULL DEFAULT '',
    recipient_email  TEXT        NOT NULL,
    status           TEXT        NOT NULL,
    attempts         INTEGER     NOT NULL DEFAULT 0,
    last_error       TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL,
    sent_at          TIMESTAMPTZ,
    CONSTRAINT notification_log_evidence_kind_key UNIQUE (evidence_id, kind)
);
