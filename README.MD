# SockTail


<p align="center">
  <img src="img/SockTail.png" width="250">
</p>

**SockTail** is a small binary that joins a device to a Tailscale network and exposes a local SOCKS5 proxy on port `1080`. It's meant for red team operations where you need network access into a target system without setting up wonky port forwards, persistent daemons, or noisy tunnels.

## Features

* SOCKS5 proxy on port `1080` over Tailscale
* Supports IPv4, IPv6, and domain addresses
* Tailscale join via `tsnet`, no external dependencies
* Auth key can be hardcoded (XOR-obfuscated for static analysis evasion) or supplied at runtime
* Fully self-contained, no configuration files or (disk writes)

This is not meant for persistence. It's a one-shot SOCKS dropper for red team ops.

<p align="center">
  <img src="https://trusted-sec.transforms.svdcdn.com/production/images/Blog-assets/SOCKS_Esteban/Fig3_Esteban.png?w=840&q=90&fm=webp&fit=max&dm=1701370766&s=7fa4e05233215a1f432a7a5d5526ef68" width="400">
</p>
<p align="center">
  <em>Image credit: <a href="https://trustedsec.com/blog/the-socks-we-have-at-home">"The SOCKS We Have at Home" by TrustedSec</a> — thanks for the memes.</em>
</p>



## Usage

```
Usage: ./SockTail [hostname] [authkey] [control-url]

  hostname:    Optional. Auto-generated if not specified
  authkey:     Optional. Uses embedded key if not specified
  control-url: Optional. Uses Tailscale default or build-time URL if not specified
  port:        Fixed at 1080

Examples:
  ./SockTail
    - Auto hostname, embedded key, default control server

  ./SockTail vpn-srv-01
    - Custom hostname, embedded key, default control server

  ./SockTail shellbox-7 tskey-auth-1fXXXXXXXXXXXXXXXXXXXXXXXXXX
    - Custom hostname and runtime auth key, default control server

  ./SockTail my-proxy tskey-auth-1fXXXXXXXXXXXXXXXXXXXXXXXXXX https://headscale.example.com
    - All custom: hostname, auth key, and control server (e.g., Headscale)
```


## How It Works

1. On startup, SockTail creates a `tsnet.Server` using the provided (or embedded) Tailscale auth key and hostname.
2. It joins the Tailnet and starts listening on `localhost:1080`.
3. SOCKS5 negotiation is handled with no authentication (standard NO\_AUTH method).
4. Any CONNECT requests (IPv4, IPv6, domain) are accepted and forwarded using `tsnet.Dial`.
5. Data is relayed bi-directionally until the connection is closed.

There’s no persistence. Once the binary exits, the connection to your Tailnet is dropped.


## Obfuscation

The project includes XOR-based obfuscation for the AuthKey to evade static detections. You can embed your auth key at build time using compiler flags - no manual editing required!

```go
var xorKey = []byte("747sg^8N0$")
```

The key will be XOR-obfuscated and embedded automatically during the build process if no key is passed on the command line.


## Build

Written in Go, no external dependencies:

```bash
# Build with embedded fallback key
make build

# Build with your custom auth key (recommended)
make build-with-key AUTH_KEY=tskey-auth-client-xxxxx-your-key

# Build with custom auth key and control server (for Headscale, etc.)
make build-with-config AUTH_KEY=tskey-auth-client-xxxxx-your-key CONTROL_URL=https://headscale.example.com

# Build for all platforms with your key
make build-all-with-key AUTH_KEY=tskey-auth-client-xxxxx-your-key

# Build for all platforms with custom control server
make build-all-with-config AUTH_KEY=tskey-auth-client-xxxxx-your-key CONTROL_URL=https://headscale.example.com
```

The build system automatically XOR-obfuscates your auth key and embeds both the key and control URL at compile time - no manual source editing required!

**Custom Control Servers**: Perfect for self-hosted Tailscale solutions like [Headscale](https://github.com/juanfont/headscale). Just specify your control server URL and the appropriate auth key for that instance.

## Notes

* Traffic is end-to-end encrypted via Tailscale/WireGuard.
* You can use this to pivot into a network, tunnel C2, or access internal services without exposing ports externally.
* Tailscale ACLs still apply. Make sure your Tailnet allows the correct access to and from the SockTail node.
