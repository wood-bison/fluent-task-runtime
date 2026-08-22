# A queue in Postgres two workers can share

A jobs table is the cheapest queue there is, and it is correct only if the
claim is atomic. The naive version reads pending rows, then updates them. In
between, a second worker reads the same rows — and the job runs twice.

`SELECT ... FOR UPDATE` fixes the race and introduces a second problem: the
other worker now *waits* for the lock instead of taking different work, so
throughput collapses to one worker. `SKIP LOCKED` is the piece that makes the
pattern usable: locked rows are passed over, not queued behind.

Do it in one statement. The check runs two workers at the same moment and looks
at three things: whether their claimed sets overlap, whether both got a full
batch, and whether the rows were actually marked.
