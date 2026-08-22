FROM golang:1.24-bookworm AS build
WORKDIR /src
ARG TARGETOS=linux
ARG TARGETARCH
COPY go.mod ./
COPY contracts ./contracts
COPY internal ./internal
COPY cmd ./cmd
RUN test -n "$TARGETARCH" || (echo 'TARGETARCH is required for a reproducible image build' >&2 && exit 1)
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/runtime ./cmd/runtime

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/runtime /runtime
USER nonroot:nonroot
EXPOSE 48227
ENTRYPOINT ["/runtime"]
