### Structure

```json
{
  "type": "chain",
  "tag": "chain-out",
  "outbounds": [
    "proxy-a",
    "proxy-b",
    "proxy-c"
  ]
}
```

### Fields

#### outbounds

==Required==

List of outbound tags that form the chain, in packet path order.

- `outbounds[0]` is the first hop (closest to the client).
- The last entry is the final exit hop.
- Minimum length is 1 (single hop is allowed but usually unnecessary). Recommended minimum is 2.

### Behaviour

Chain is a first-class outbound that makes multi-hop proxying explicit and user-friendly.

At configuration load time the compiler:

1. Validates that every hop exists and is not another chain (nesting is currently forbidden).
2. Rejects hops that already have a `detour` set or that are `direct` when used as non-final hops.
3. Creates private synthetic intermediate outbounds that wire the detours automatically.
4. Exposes a single entry point (the first hop) under the chain's own tag.

The original outbounds remain unchanged and can still be used independently. Traffic policy continues to be controlled solely by route rules.

### Restrictions (Fail-Closed)

- Nested chains are not supported.
- A hop that already contains a `detour` cannot be used inside a chain.
- `direct` cannot be used as a non-final hop.
- Synthetic internal tags are reserved; user-defined tags that collide with them cause a clear load-time error.

### Example

```json
{
  "outbounds": [
    {
      "type": "shadowsocks",
      "tag": "ss-entry",
      "server": "entry.example.com",
      "server_port": 8388,
      "method": "aes-256-gcm",
      "password": "password"
    },
    {
      "type": "vless",
      "tag": "vless-exit",
      "server": "exit.example.com",
      "server_port": 443,
      "uuid": "uuid",
      "tls": { "enabled": true, "server_name": "exit.example.com" }
    },
    {
      "type": "chain",
      "tag": "my-chain",
      "outbounds": ["ss-entry", "vless-exit"]
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "final": "my-chain"
  }
}
```
