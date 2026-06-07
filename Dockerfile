# Build a static binary with a current Go toolchain.
FROM golang:1.24 AS builder

WORKDIR /src
COPY . .

# CGO disabled => fully static binary that runs on a minimal base image.
# Build from the vendored dependencies for reproducible, offline builds.
RUN CGO_ENABLED=0 go build -mod=vendor -o /out/server ./cmd/server

# Minimal runtime: distroless static ships ca-certificates + tzdata, runs as a
# non-root user, and contains no shell or package manager.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/server /server

EXPOSE 8080
ENTRYPOINT ["/server"]
