# Chain Outbound (Fork Feature)

This document describes the native Chain outbound added in this fork.

## Goal

Make multi-hop (chained) proxying a first-class, declarative, and user-friendly feature of sing-box, while preserving full compatibility with the official core and enabling long-term upstream synchronization.

## Configuration

```json
{
  "type": "chain",
  "tag": "my-chain",
  "outbounds": ["hop1", "hop2", "hop3"]
}
```

- Order is **packet path order**: `outbounds[0]` is the first hop closest to the client.
- The last entry is the final exit hop.
- Minimum length is 1 (single hop is accepted but rarely useful).

## How it works

At configuration load time (`CompileChainOutbounds`):

1. Validates tags, rejects duplicates, self-references, nested chains, and hops that already have a `detour`.
2. Rejects `direct` as a non-final hop.
3. Creates private synthetic intermediate outbounds (with reserved tags) and automatically sets the correct `detour` values.
4. The original user-defined outbounds remain completely unmodified and can still be used independently.
5. The chain itself exposes only the entry point under its own tag.

## Restrictions (Fail-Closed)

- Nested chains are currently not supported.
- Intermediate hops must support `DialerOptions` (most protocol outbounds do).
- Hops that already contain a `detour` cannot be reused inside a chain.
- Synthetic internal tags are reserved; collisions with user tags produce a clear error.

## Benefits over manual `detour`

- Explicit and readable configuration.
- No need to mutate shared outbounds.
- Clear ownership of the chain topology.
- Easier for clients/GUIs to display and manage multi-hop paths.
- Safer (Fail-Closed) error handling.

## Upstream synchronization

Changes are localized primarily to:

- `option/chain.go` + `option/chain_compile.go`
- `protocol/chain/`
- Registration in `include/registry.go` and constant in `constant/proxy.go`
- Call site in `option/options.go`

When merging official updates, re-run the existing unit tests in `option/chain_compile_test.go`. Conflicts, if any, will surface as compile or test failures rather than silent behavioral changes.

## Examples

See `examples/chain-basic.json` and `examples/chain-ss-vless.json`.
