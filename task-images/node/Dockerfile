# The Node image every code-workspace task in this sub-project runs against.
# Node 24 to match the plan's baseline (ADR-0006), independent of whatever
# Pin the toolchain inside the task image; the host toolchain is irrelevant.
FROM node:24-slim

RUN mkdir -p /opt/harness
COPY harness/run.js /opt/harness/run.js

# /solution (the learner's workspace), /hidden-tests (server-only tests), and
# /output (where results.json is written) are bind-mounted at `docker run`
# time by container.executor — nothing to declare here.
WORKDIR /solution
