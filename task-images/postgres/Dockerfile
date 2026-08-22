# A real PostgreSQL task image. The runner still provides the outer sandbox
# (read-only solution, no network, bounded CPU/memory); this image starts a
# short-lived local server on a Unix socket so EXPLAIN is genuine PostgreSQL.
FROM postgres:17

RUN mkdir -p /opt/harness
COPY harness/run.sh /opt/harness/run.sh

# The upstream image's entrypoint is for a long-lived database container. A
# task owns its own short-lived server, so let ContainerExecutor invoke the
# declared check command directly.
ENTRYPOINT []
# /solution (learner files), /hidden-tests (server-only tests), and /output
# are bind-mounted by container.executor at runtime.
WORKDIR /solution
