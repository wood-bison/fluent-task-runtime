CREATE TABLE jobs (
  id bigint PRIMARY KEY,
  status text NOT NULL DEFAULT 'pending',
  claimed_by text,
  payload text NOT NULL
);

INSERT INTO jobs (id, payload)
SELECT series, 'job-' || series
FROM generate_series(1, 200) AS series;

-- A partial index over the hot slice only: the pending rows are the ones
-- workers scan, and indexing the finished ones would grow the index for
-- nothing.
CREATE INDEX jobs_pending_idx ON jobs (id) WHERE status = 'pending';

ANALYZE jobs;
