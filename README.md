# pzmonitor

[![CI](https://github.com/MarioMoura/pzmonitor/actions/workflows/ci.yml/badge.svg)](https://github.com/MarioMoura/pzmonitor/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/MarioMoura/pzmonitor)](https://github.com/MarioMoura/pzmonitor/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Prometheus exporter for Project Zomboid dedicated servers. Collects server metrics via RCON and exposes them on a `/metrics` endpoint.

![Grafana dashboard — server overview](docs/dashboard-overview.png)

![Grafana dashboard — players and world](docs/dashboard-players.png)

A ready-to-import Grafana dashboard is included at [`grafana/dashboard.json`](grafana/dashboard.json).

## Metrics

- **Server health**: FPS, JVM memory (used/total/max), average update period
- **Players & world**: players online, zombies (loaded/simulated/total), loaded cells, animal instances
- **Daily events**: zombies killed, players killed (by zombie/player/fire), zombified players, burned corpses
- **Network**: bytes sent/received per second, packet loss
- **Operational**: scrape duration, server up/down

## Installation

Download the binary for your platform from the [Releases](https://github.com/MarioMoura/pzmonitor/releases) page, then:

```bash
chmod +x pzmonitor
./pzmonitor
```

Or build from source:

```bash
go install github.com/MarioMoura/pzmonitor@latest
```

### Ansible

The repo ships an Ansible collection (`ansible/`) with a `pzmonitor` role that
downloads the release binary and installs a systemd unit.

Add it to your `requirements.yml` (pin the tag to the release you want):

```yaml
collections:
  - name: https://github.com/MarioMoura/pzmonitor.git#ansible
    type: git
    version: v0.1.6
```

Install and use it:

```bash
ansible-galaxy collection install -r requirements.yml
```

```yaml
- hosts: zomboid
  become: true
  roles:
    - role: mariomoura.pzmonitor.pzmonitor
      vars:
        pzmonitor_rcon_password: "{{ rcon_password }}"
```

Useful variables (see `ansible/roles/pzmonitor/defaults/main.yml` for all):

| Variable | Default | Description |
|---|---|---|
| `pzmonitor_version` | release version | Release to download |
| `pzmonitor_user` | `pzmonitor` | User that runs the service (created unless `pzmonitor_create_user: false`) |
| `pzmonitor_install_dir` | `/opt/pzmonitor` | Where the binary lives |
| `pzmonitor_port` | `9103` | Metrics listen port |
| `pzmonitor_rcon_host` / `pzmonitor_rcon_port` | `127.0.0.1` / `27015` | RCON target |
| `pzmonitor_after_unit` | `""` | systemd unit to start after, e.g. `zomboid.service` |
| `pzmonitor_env` | `{}` | Extra `PZMONITOR_*` environment variables |

## Configuration

All configuration is done via environment variables:

| Variable | Default | Description |
|---|---|---|
| `PZMONITOR_RCON_HOST` | `127.0.0.1` | RCON server host |
| `PZMONITOR_RCON_PORT` | `27015` | RCON server port |
| `PZMONITOR_RCON_PASSWORD` | *(required)* | RCON password |
| `PZMONITOR_LISTEN_ADDR` | `:9101` | Address for the HTTP metrics server |
| `PZMONITOR_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

Copy `.env.example` as a reference:

```bash
cp .env.example .env
```

## Prometheus

Add a scrape job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: pzmonitor
    static_configs:
      - targets: ["localhost:9101"]
```

## Endpoints

- `GET /metrics` - Prometheus metrics
- `GET /healthz` - Health check
