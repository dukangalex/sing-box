# Experimental local Fortinet XML candidate

Base: github.com/sagernet/sing-openconnect v0.1.1-0.20260929151226-29757281a247, copied from the matching module cache. The root go.mod replaces that exact module with this directory. This is experimental, not an upstream or release assertion.

The parser retains the application's existing 16 MiB HTTP body limit and has separate experimental markup, attribute, semantic and nesting budgets. These are application policy, not established Fortinet protocol maxima; they may reject valid configurations. Synthetic compatibility and Linux host tests do not establish real-server compatibility, Android memory behavior or production acceptance. Preserve the upstream license and complete those checks before promotion.
