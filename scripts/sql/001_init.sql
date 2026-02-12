-- Schema for chunk_requests table used by internal/chunklog DBEvent.
-- TimescaleDB / PostgreSQL.

CREATE TABLE IF NOT EXISTS requests (
    time                  TIMESTAMPTZ       NOT NULL,
    path                  TEXT              NOT NULL,
    ip                    INET,
    referer               VARCHAR(255),
    sid                   UUID              NOT NULL,
    uid                   UUID              NOT NULL,
    chunk_codec           SMALLINT,
    chunk_quality         SMALLINT,
    chunk_size            BIGINT,
    chunk_duration        INTEGER,
    chunk_timestamp       TIMESTAMPTZ,
    chunk_sequence        BIGINT,
    ua_browser            VARCHAR(255),
    ua_browser_version    VARCHAR(255),
    ua_device             VARCHAR(255),
    ua_os                 VARCHAR(255),
    ua_is_desktop         BOOLEAN,
    ua_is_mobile          BOOLEAN,
    ua_is_tablet          BOOLEAN,
    ua_is_tv              BOOLEAN,
    ua_is_bot             BOOLEAN,
    ua_is_android         BOOLEAN,
    ua_is_ios             BOOLEAN,
    ua_is_windows         BOOLEAN,
    ua_is_linux           BOOLEAN,
    ua_is_mac             BOOLEAN,
    ua_is_openbsd         BOOLEAN,
    ua_is_chromeos        BOOLEAN,
    ua_is_chrome          BOOLEAN,
    ua_is_firefox         BOOLEAN,
    ua_is_safari          BOOLEAN,
    ua_is_edge            BOOLEAN,
    ua_is_opera           BOOLEAN,
    ua_is_samsung_browser BOOLEAN,
    ua_is_vivaldi         BOOLEAN,
    ua_is_yandex_browser  BOOLEAN
) WITH (
    tsdb.hypertable
);

---- create above / drop below ----

DROP TABLE IF EXISTS requests;