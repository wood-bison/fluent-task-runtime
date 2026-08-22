# Deterministic Go profile for bounded interview tasks.
FROM golang:1.24-bookworm

RUN mkdir -p /opt/harness
# The runner intentionally mounts /tmp noexec. Go's compiler creates and then
# executes a temporary test binary, so keep its bounded scratch directories in
# the writable output mount instead of weakening the noexec boundary.
ENV GOCACHE=/output/go-cache GOTMPDIR=/output/go-tmp TMPDIR=/output/tmp
COPY harness/run.sh /opt/harness/run.sh
RUN chmod 0555 /opt/harness/run.sh
WORKDIR /solution
