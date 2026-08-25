FROM golang:1.24-bookworm AS build
WORKDIR /src
ARG TARGETOS=linux
ARG TARGETARCH
COPY go.mod go.sum ./
COPY contracts ./contracts
COPY internal ./internal
COPY cmd ./cmd
COPY tasks ./tasks
COPY task-families ./task-families
COPY releases ./releases
RUN test -n "$TARGETARCH" || (echo 'TARGETARCH is required for a reproducible image build' >&2 && exit 1)
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/runtime ./cmd/runtime

FROM docker:27-cli
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
