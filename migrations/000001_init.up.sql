CREATE TYPE notification_channel  AS ENUM ('sms', 'email', 'push');
CREATE TYPE notification_status   AS ENUM ('pending', 'queued', 'processing', 'delivered', 'failed', 'cancelled');
CREATE TYPE notification_priority AS ENUM ('high', 'normal', 'low');

CREATE TABLE notifications (
    id              UUID PRIMARY KEY,
    batch_id        UUID,
    recipient       VARCHAR(255) NOT NULL,
    channel         notification_channel  NOT NULL,
    content         TEXT NOT NULL,
    priority        notification_priority NOT NULL DEFAULT 'normal',
    status          notification_status   NOT NULL DEFAULT 'pending',
    idempotency_key VARCHAR(255) UNIQUE,
    provider_msg_id VARCHAR(255),
    retry_count     INT         NOT NULL DEFAULT 0,
    scheduled_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- channel-only filter (List API)
CREATE INDEX idx_notifications_channel     ON notifications(channel);
-- batch-only filter (List API)
CREATE INDEX idx_notifications_batch_id    ON notifications(batch_id);
-- date-range filter without status (List API)
CREATE INDEX idx_notifications_created_at  ON notifications(created_at DESC);
-- status filter + ORDER BY created_at (List API, Archive worker)
CREATE INDEX idx_notifications_status_created  ON notifications(status, created_at DESC);
-- status + channel combined filter (List API)
CREATE INDEX idx_notifications_status_channel  ON notifications(status, channel);
-- scheduler: only pending rows that have a scheduled time
CREATE INDEX idx_notifications_scheduled   ON notifications(scheduled_at)
    WHERE scheduled_at IS NOT NULL AND status = 'pending';

CREATE TABLE notifications_archive (
    LIKE notifications INCLUDING ALL
);

CREATE INDEX idx_archive_created_at ON notifications_archive(created_at DESC);
CREATE INDEX idx_archive_status     ON notifications_archive(status);
