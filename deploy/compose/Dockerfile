FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY contracts ./contracts
COPY internal ./internal
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o /out/runtime ./cmd/runtime

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/runtime /runtime
USER nonroot:nonroot
EXPOSE 48227
ENTRYPOINT ["/runtime"]
