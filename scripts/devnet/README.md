# 2-Node Devnet — Multi-node Smoke Tests

This directory contains the docker-compose configuration and helper scripts for running a 2-validator local devnet that satisfies the §4.28–§4.30 multi-node smoke acceptance items.

## Files

- `../../docker-compose.devnet.yml` — 2-node docker-compose
- `init-node1.sh` — genesis-creating validator init
- `init-node2.sh` — joining peer init (pulls genesis from node-1)
- `smoke.sh` — runs §4.28 (block production), §4.29 (slashing), §4.30 (bridge sidecar)

## Boot

```bash
cd qorechain-core/
docker compose -f docker-compose.devnet.yml up --build -d

# Wait for both nodes to report healthy
docker compose -f docker-compose.devnet.yml ps

# Verify finality
curl -s http://localhost:26657/status | jq .result.sync_info
curl -s http://localhost:26757/status | jq .result.sync_info
```

## Run smoke tests

```bash
./scripts/devnet/smoke.sh
```

Exit codes:
- `0` — all pass
- `1` — §4.28 (block production / finality) failed
- `2` — §4.29 (slashing) failed
- `3` — §4.30 (bridge sidecar) failed

## Notes

- `timeout_commit` is set to 1s for fast feedback (production default is 5s).
- Node-2 uses the address-book and `persistent_peers` config to dial node-1; the genesis is pulled over the JSON-RPC `/genesis` endpoint.
- §4.30 (bridge sidecar) requires `anvil` (Foundry) on the host or skips with a warning. A full attestation round trip needs a deployed bridge contract on Anvil and the Ethereum sidecar container — out of smoke-test scope; see `docs/BRIDGE.md` for the full integration test path.

## Tear down

```bash
docker compose -f docker-compose.devnet.yml down -v
```

The `-v` flag removes the node home volumes; on the next boot, node-1 will create a fresh genesis.
