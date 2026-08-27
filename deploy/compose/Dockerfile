FROM golang:1.24-bookworm@sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac AS build
WORKDIR /src
ARG TARGETOS=linux
ARG TARGETARCH
ARG SOURCE_REVISION=unknown
RUN printf '%s' "$SOURCE_REVISION" | grep -Eq '^[0-9a-f]{40}$' || (echo 'SOURCE_REVISION must be a 40-character Git SHA' >&2 && exit 1)
COPY go.mod go.sum ./
COPY contracts ./contracts
COPY internal ./internal
COPY cmd ./cmd
COPY tasks ./tasks
COPY task-families ./task-families
COPY releases ./releases
RUN test -n "$TARGETARCH" || (echo 'TARGETARCH is required for a reproducible image build' >&2 && exit 1)
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/runtime ./cmd/runtime

FROM docker:27-cli@sha256:851f91d241214e7c6db86513b270d58776379aacc5eb9c4a87e5b47115e3065c
ARG SOURCE_REVISION=unknown
LABEL org.opencontainers.image.revision="$SOURCE_REVISION"
COPY --from=build /out/runtime /runtime
COPY --from=build /src/tasks /opt/tasks
COPY --from=build /src/task-families /opt/task-families
COPY --from=build /src/releases /opt/releases
ENV RUNTIME_TASKS_ROOT=/opt/tasks
ENV RUNTIME_TASK_FAMILY_MANIFEST=/opt/task-families/manifest.json
EXPOSE 48227
ENTRYPOINT ["/runtime"]
