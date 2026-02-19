# Stage 1: Build the Go binary
FROM golang:1.26-bookworm AS builder

WORKDIR /build

# Copy go modules first (caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Copy pre-compiled PQC library
COPY lib/linux_amd64/libqorepqc.so /usr/local/lib/

# Build with CGO enabled (for Rust FFI)
ENV CGO_ENABLED=1
ENV LD_LIBRARY_PATH=/usr/local/lib
RUN go build -tags "netgo ledger" -ldflags "-w -s" -o /build/qorechaind ./cmd/qorechaind

# Stage 2: Minimal runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates curl jq && rm -rf /var/lib/apt/lists/*

# Copy binary and PQC library
COPY --from=builder /build/qorechaind /usr/local/bin/
COPY --from=builder /usr/local/lib/libqorepqc.so /usr/local/lib/
RUN ldconfig

# Copy init scripts
COPY scripts/ /scripts/
RUN chmod +x /scripts/*.sh

EXPOSE 26657 26656 1317 9090 26660

ENTRYPOINT ["/scripts/entrypoint.sh"]
