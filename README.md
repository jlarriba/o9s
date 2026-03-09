```
  ___  ___  __
 / _ \/ _ \/ _/
 \___/\_, /\__/
     /___/
```

# o9s — OpenStack TUI

A terminal UI for OpenStack, inspired by [k9s](https://k9scli.io/). Browse, inspect, and manage OpenStack resources interactively.

## Install

```bash
go build -o o9s .
```

## Usage

```bash
# Authenticate via clouds.yaml
./o9s --cloud mycloud

# Or via OS_* environment variables
export OS_AUTH_URL=... OS_USERNAME=... OS_PASSWORD=... OS_PROJECT_NAME=...
./o9s
```

## Supported Resources

server, network, subnet, router, volume, image, flavor, keypair, security group, floating IP, port, project

## Key Bindings

| Key | Action |
|-----|--------|
| `:` | Command bar — type a resource name to switch views |
| `Enter` | Inspect selected resource |
| `Escape` / `q` | Back / quit |
| `r` | Reload |
| `Ctrl-d` | Delete (with confirmation) |
| `j` / `l` | Start / stop server |
| `0-9` | Switch project |
| `F1-F5` | Quick-switch: server, network, subnet, volume, router |

## Features

- Auto-refreshing resource tables (30s)
- Color-coded status columns
- Quota usage bars (vCPUs, RAM, volumes, storage)
- Live CPU%, memory%, and disk% metrics for servers (via Aetos/Prometheus)
- Multi-project support with instant switching
- Works for non-admin users
