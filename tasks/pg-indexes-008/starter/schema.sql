CREATE TABLE orders (
  id bigint PRIMARY KEY,
  customer_id integer NOT NULL,
  created_at timestamptz NOT NULL,
  total_cents integer NOT NULL,
  status text NOT NULL
);

INSERT INTO orders (id, customer_id, created_at, total_cents, status)
SELECT
  series,
  ((series - 1) % 500) + 1,
  timestamptz '2025-01-01 00:00:00+00' + (series * interval '1 minute'),
  100 + (series % 10000),
  CASE WHEN series % 100 = 0 THEN 'refunded' ELSE 'paid' END
FROM generate_series(1, 50000) AS series;

ANALYZE orders;
