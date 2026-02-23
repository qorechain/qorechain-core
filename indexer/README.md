# Block Indexer

The Block Indexer is a WebSocket-based service that listens for new blocks on QoreChain,
processes transactions, and stores indexed data in PostgreSQL for fast querying.

## Features

- Real-time block and transaction indexing via WebSocket
- PostgreSQL storage with optimized schema
- Transaction type classification and metadata extraction
- PQC signature tracking and algorithm statistics

## Build

The indexer binary is distributed as a pre-built Docker image.
See the [QoreChain documentation](https://qorechain.io/docs) for deployment instructions.
