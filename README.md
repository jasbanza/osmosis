# Osmosis -- Snapshot Ingestor Branch

> **This is a special-purpose branch: `osmosis-snapshot-for-sqs-ingest`**
>
> This branch provides a patched `osmosisd` binary that starts from a mainnet snapshot, pushes
> **all** pool data to SQS via gRPC on the very first committed block, and then **halts automatically**.
> It is designed to work with a companion SQS fork for offline reproduction and testing of routing
> fixes (e.g. OSMO-53).
>
> **Key differences from upstream `osmosis-labs/osmosis`:**
>
> - **Node sync check removed** -- the block processor skips the "is node syncing" gate so pool
>   data is ingested immediately, even while the node is catching up from a snapshot.
> - **Auto-halt after SQS push** -- `os.Exit(0)` is called after the first successful pool data
>   push, so the node stops automatically once its job is done.
> - **Pre-flight gRPC connectivity check** -- on startup, the binary verifies SQS is reachable
>   (5-second timeout). If not, it panics immediately to avoid wasting time syncing blocks.
>
> **Related repositories:**
>
> - SQS snapshot fork: [jasbanza/sqs (`jason/sqs-for-snapshot-and-cached-coingecko`)](https://github.com/jasbanza/sqs/tree/jason/sqs-for-snapshot-and-cached-coingecko)
> - Upstream Osmosis: [osmosis-labs/osmosis](https://github.com/osmosis-labs/osmosis)
> - Upstream SQS: [osmosis-labs/sqs](https://github.com/osmosis-labs/sqs)
>
> **Important: SQS must be running before osmosisd starts.** The pre-flight check will panic if
> the SQS gRPC endpoint is not reachable. See Steps 5-6 below.

---

## Files changed

Only two files are modified from upstream:

| File | Change |
|------|--------|
| `ingest/sqs/service/blockprocessor/full_sqs_block_process_strategy.go` | Removed the node sync check; added structured pool count logging; added auto-halt (`os.Exit(0)`) after successful SQS push |
| `app/app.go` | Added pre-flight gRPC connectivity check -- panics immediately if SQS isn't reachable |

The rest of the codebase is identical to upstream `osmosis-labs/osmosis`.

---

## Prerequisites

- An Osmosis snapshot (any age) restored to `~/.osmosisd/data/`
- The companion SQS fork (snapshot branch) -- see Related repositories above
- Both on the same machine (or adjust addresses accordingly)
- Go 1.21+ installed

---

## Step 1: Build the patched binary

```bash
git clone https://github.com/jasbanza/osmosis.git
cd osmosis
git checkout osmosis-snapshot-for-sqs-ingest
go install ./cmd/osmosisd
```

Verify:

```bash
osmosisd version
```

## Step 2: Restore your snapshot

If you haven't already, restore your snapshot into `~/.osmosisd/data/`. Check your current height:

```bash
curl -sS http://localhost:26657/status 2>/dev/null \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['result']['sync_info']['latest_block_height'])" \
  || echo "osmosisd not running yet - that's fine"
```

## Step 3: Add seeds to `config.toml`

Edit `~/.osmosisd/config/config.toml`, find the `[p2p]` section, and set:

```toml
seeds = "20e1000e88125698264454a884812746c2eb4807@seeds.lavenderfive.com:12556,ade4d8bc8cbe014af6ebdf3cb7b1e9ad36f412c0@seeds.polkachu.com:12556,ebc272824924ea1a27ea3183dd0b9ba713494f83@osmosis-mainnet-seed.autostake.com:26716,3cc024d1c760c9cd96e6413abaf3b36a8bdca58e@seeds.goldenratiostaking.net:1630,e891d42c31064fb7e0d99839536164473c4905c2@seed-osmosis.freshstaking.com:31656"
```

## Step 4: Enable SQS ingestion in `app.toml`

Edit `~/.osmosisd/config/app.toml` and add/update the SQS section:

```toml
[osmosis-sqs]
is-enabled = true
grpc-ingest-address = [localhost:50051]
grpc-ingest-max-call-size-bytes = 52428800
```

## Step 5: Start SQS first (required before osmosisd)

**SQS must be running and listening on port 50051 before you start osmosisd.** The patched binary performs a pre-flight gRPC connectivity check on startup and will panic if SQS is not reachable.

```bash
cd /root/sqs   # or wherever your SQS source is
go run ./...

# or 
make dev
```

Wait until you see it's listening on port 50051, then proceed to Step 6.

## Step 6: Start the patched osmosisd

```bash
osmosisd start
```

You should see:

1. `SQS pre-flight check passed: localhost:50051 is reachable`
2. The node syncing / catching up to the network
3. On the first committed block: pool extraction logs with counts (concentrated, cfmm, cosmwasm)
4. `SQS INGEST COMPLETE` / `All pool data pushed successfully. Halting.`
5. The process exits automatically (`os.Exit(0)`)

## Step 7: Verify SQS received the data

Check SQS has the pool data:

```bash
curl -sS http://localhost:9092/pools | python3 -m json.tool | head -50
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Panic: `SQS pre-flight check failed: cannot reach SQS gRPC at ...` | SQS is not running or not listening on the configured port | Start SQS first (Step 5), confirm port 50051 is open |
| Panic on startup with app version / upgrade handler errors | Snapshot is from a height incompatible with this binary (v31) | Use a recent snapshot compatible with the v31 module version |
| Node hangs at "Searching for peers..." | No seeds configured or firewall blocking P2P | Verify seeds in `config.toml` (Step 3), check firewall allows outbound on port 26656 |
| gRPC errors during pool push | Message size exceeds limit | Increase `grpc-ingest-max-call-size-bytes` in `app.toml` |
| SQS reports CoinGecko rate limit errors | SQS branch does not have the extended caching fix | Use the patched SQS branch with extended CoinGecko cache TTL |

---

## Original upstream README

For the full Osmosis documentation, see the [upstream repository](https://github.com/osmosis-labs/osmosis).

Osmosis is the largest DEX in the Cosmos ecosystem, serving as a source of liquidity for over 50 sovereign blockchains connected via IBC. It features concentrated liquidity, Superfluid Staking, Protocol Revenue, and a cross-chain trading suite. For system requirements, build instructions, and full documentation, refer to the upstream repo.
