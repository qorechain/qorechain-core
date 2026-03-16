# Stage 1: Build the Go binary
FROM golang:1.26-bookworm AS builder

RUN apt-get update && apt-get install -y build-essential && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go modules first (caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with CGO enabled (required for go-ethereum secp256k1)
# Public build — no PQC library, no full tags
ENV CGO_ENABLED=1
RUN go build -tags "netgo ledger" -ldflags "-w -s" -o /build/qorechaind ./cmd/qorechaind

# Stage 2: Minimal runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates curl jq && rm -rf /var/lib/apt/lists/*

# Copy binary
COPY --from=builder /build/qorechaind /usr/local/bin/

# Copy init scripts
COPY scripts/ /scripts/
RUN chmod +x /scripts/*.sh

# Create non-root user
RUN useradd -r -u 1000 -d /home/qorechaind -s /sbin/nologin qorechaind && \
    mkdir -p /home/qorechaind/.qorechaind && \
    chown -R qorechaind:qorechaind /home/qorechaind

USER qorechaind
WORKDIR /home/qorechaind

# QoreChain RPC, P2P, REST, gRPC, Prometheus, EVM JSON-RPC, EVM WS
EXPOSE 26657 26656 1317 9090 26660 8545 8546

ENTRYPOINT ["/scripts/entrypoint.sh"]
