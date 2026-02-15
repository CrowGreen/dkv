# dkv

Distributed key-value store written in Go with gRPC, TLS, and encrypted storage.

## Features

- **gRPC API** with Get, Put, Delete, BatchPut, and Watch (server-sent streaming)
- **Write-ahead logging** for crash recovery — binary WAL format with fsync
- **AES-256-GCM encrypted snapshots** for persistent storage at rest
- **TLS 1.3** mutual authentication for all RPCs
- **API-key auth** via gRPC interceptors with per-client identity
- **Raft-inspired replication** with leader election and log replication
- **Concurrent** request handling via goroutines and channels
- **Watch** support for real-time key change notifications with prefix filtering

## Architecture

```
Client (CLI/gRPC)
    |
    | TLS + API Key
    v
gRPC Server (interceptors: auth, logging)
    |
    +---> KV Store (sync.RWMutex, in-memory map)
    |         |
    |         +---> WAL (binary log, fsync)
    |         +---> Snapshot (AES-GCM encrypted, periodic)
    |         +---> Watchers (channel-based pub/sub)
    |
    +---> Replication (Raft-inspired)
              |
              +---> AppendEntries RPC
              +---> RequestVote RPC
```

## Quick start

```bash
# generate TLS certs (optional)
make certs

# build
make build

# run server
./bin/dkv-server --port 5050 --api-key my-secret-key

# in another terminal, use client
./bin/dkv-client --key my-secret-key put name alice
./bin/dkv-client --key my-secret-key get name
./bin/dkv-client --key my-secret-key watch user:
```

## Run tests

```bash
make test
# or with race detector
go test ./... -race -v
```

## Design decisions

**Why raw binary WAL?** Faster than JSON/protobuf for append-only writes. Each entry is `[length][op][keyLen][key][value]` — no parsing overhead on replay.

**Why AES-GCM over AES-CBC?** GCM provides authenticated encryption (integrity + confidentiality) in one pass. Detects tampering without a separate HMAC.

**Why not full Raft?** The replication layer implements the core election and log replication mechanics but skips log compaction and dynamic membership for simplicity. Enough to demonstrate the pattern without the operational complexity.
