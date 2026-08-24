CREATE TABLE rate_limit_events (
  id bigserial PRIMARY KEY,
  client_id text NOT NULL,
  occurred_at timestamptz NOT NULL
);

CREATE INDEX rate_limit_events_client_time_idx
  ON rate_limit_events (client_id, occurred_at DESC);

INSERT INTO rate_limit_events (client_id, occurred_at)
SELECT 'full', timestamptz '2025-01-01 00:00:00+00' + (series * interval '1 second')
FROM generate_series(0, 4) AS series;

ANALYZE rate_limit_events;
