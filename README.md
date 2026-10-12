# sing-box

The universal proxy platform.

[![Build Status](https://github.com/SagerNet/sing-box/actions/workflows/build.yml/badge.svg)](https://github.com/SagerNet/sing-box/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/SagerNet/sing-box)](https://goreportcard.com/report/github.com/SagerNet/sing-box)
[![License](https://img.shields.io/github/license/SagerNet/sing-box)](https://github.com/SagerNet/sing-box/blob/main/LICENSE)

## Documentation

https://sing-box.sagernet.org

## License

```
Copyright (C) 2022 by Sager Network Inc All Rights Reserved.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.

In addition, no derivative work may use the name or imply association
with this application without prior consent.
```

## Chain Outbound (Fork Feature)

This fork adds a native `chain` outbound type that makes multi-hop proxying explicit and configuration-friendly.

```json
{
  "type": "chain",
  "tag": "my-chain",
  "outbounds": ["entry-proxy", "exit-proxy"]
}
```

- Order is packet path order (`outbounds[0]` is the first hop).
- The compiler automatically wires private intermediate hops with the correct `detour` values.
- Original outbounds remain unchanged and can still be used independently.
- Nested chains and hops that already have a `detour` are rejected at load time (Fail-Closed).

See [documentation](docs/configuration/outbound/chain.md) for full details.
