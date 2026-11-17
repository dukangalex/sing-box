# Local sing-openconnect security patch

This directory is copied from github.com/sagernet/sing-openconnect
v0.1.1-0.20260929151226-29757281a247 (license included). The kernel
replaces that exact module with this local copy. The only production source
change is the Fortinet server-configuration XML depth precheck in
fortinet_config.go; the zz_* test files are added locally.

The response is untrusted. Its existing 16 MiB body bound does not bound
recursive XML depth. This precheck rejects nesting beyond 256 before calling
the recursive decoder. The limit is a defensive compatibility tradeoff, not
a published Fortinet protocol limit: servers with legitimate configurations
nested more than 256 levels will fail. An adapted public response shape and
boundaries pass host tests; server interoperability and Android runtime still
require validation.
