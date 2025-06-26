# SocksTail


<p align="center">
  <img src="" width="550">
</p>

**SocksTail** is a small binary that joins a device to a Tailscale network and exposes a local SOCKS5 proxy on port `1080`. It's meant for red team operations where you need network access into a target system without setting up wonky port forwards, persistent daemons, or noisy tunnels.

---
## Features

* SOCKS5 proxy on port `1080` over Tailscale
* Supports IPv4, IPv6, and domain addresses
* Tailscale join via `tsnet`, no external dependencies
* Auth key can be hardcoded (XOR-obfuscated for static analysis evasion) or supplied at runtime
* Fully self-contained, no configuration files or (disk writes)

This is not meant for persistence. It's a one-shot SOCKS dropper for red team ops.

---

## Usage

```
Usage: ./sockstail [hostname] [authkey]

  hostname: Optional. If blank, a random one will be generated
  authkey : Optional. If blank, will use the built-in obfuscated key

Port is fixed at 1080.

Examples:
  ./sockstail
    - Random hostname, embedded key

  ./sockstail vpn-srv-01
    - Custom hostname, embedded key

  ./sockstail shellbox-7 tskey-auth-1fXXXXXXXXXXXXXXXXXXXXXXXXXX
    - Custom hostname and runtime-supplied auth key
```

---

## How It Works

1. On startup, SocksTail creates a `tsnet.Server` using the provided (or embedded) Tailscale auth key and hostname.
2. It joins the Tailnet and starts listening on `localhost:1080`.
3. SOCKS5 negotiation is handled with no authentication (standard NO\_AUTH method).
4. Any CONNECT requests (IPv4, IPv6, domain) are accepted and forwarded using `tsnet.Dial`.
5. Data is relayed bi-directionally until the connection is closed.

There’s no persistence. Once the binary exits, the connection to your Tailnet is dropped.

---

## Obfuscation

The project includes some basic XOR-based obfuscation to evade static detections:

```go
var xorKey = []byte("747sg^8N0$")
```

Use `make obfuscate-key KEY=tskey-auth-client-xxxxx-your-key` to generate a byte array and embed it into the `obfuscatedAuthKey` variable.

The key will be deobfuscated at runtime if no key is passed on the command line.

---

## Build

Written in Go, no external dependencies:

```bash
# 1. Obfuscate your key
make obfuscate-key KEY=tskey-auth-client-xxxxx-your-key

# 2. Update main.go with the generated obfuscatedAuthKey

# 3. Build  binaries
make build
```

## Notes

* Traffic is end-to-end encrypted via Tailscale/WireGuard.
* You can use this to pivot into a network, tunnel C2, or access internal services without exposing ports externally.
* Tailscale ACLs still apply. Make sure your Tailnet allows the correct access to and from the SocksTail node.
