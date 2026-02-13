-- Schema for sessions table used by internal/chunklog SessionTracker.
-- PostgreSQL.

CREATE TABLE IF NOT EXISTS sessions (
    sid                   UUID              NOT NULL,
    uid                   UUID              NOT NULL,
    source                SMALLINT          NOT NULL,
    mount                 VARCHAR(255),
    start_time            TIMESTAMPTZ       NOT NULL,
    end_time              TIMESTAMPTZ       NOT NULL,
    duration              INTERVAL,
    total_bytes           BIGINT            NOT NULL DEFAULT 0,
    codec                 SMALLINT,
    quality               SMALLINT,
    ip                    INET,
    referer               VARCHAR(255),
    user_agent            TEXT,
    ua_browser            VARCHAR(255),
    ua_browser_version    VARCHAR(255),
    ua_device             VARCHAR(255),
    ua_os                 VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS listeners_total (
    timestamp             TIMESTAMPTZ       NOT NULL,
    source                SMALLINT          NOT NULL,
    mount                 VARCHAR(255),
    count                 BIGINT            NOT NULL DEFAULT 0
);

---- create above / drop below ----

DROP TABLE IF EXISTS sessions;

DROP TABLE IF EXISTS listeners_total;