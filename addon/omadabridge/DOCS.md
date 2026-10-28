# OmadaBridge add-on

Runs [RCooLeR/omada_exporter](https://github.com/RCooLeR/omada_exporter) ("OmadaBridge") inside
Home Assistant. It polls a TP-Link Omada controller and publishes MQTT Discovery entities
(devices, ports, PoE, clients, WAN/ISP, VPN, radios). Monitoring only: device controls are never enabled.

## Requirements
- Mosquitto broker add-on + MQTT integration (the add-on picks the broker up from the Supervisor automatically).
- An Omada local user (Viewer role is enough) with 2FA disabled — `omada_user` / `omada_pass`.
- Optional but recommended: an Open API app (Client Credentials, Viewer) — `omada_client_id` / `omada_secret_id`.
  Without it WAN, ISP, VPN and detailed client metrics are skipped.
- Cards: the qoxi Mushroom fork (`mushroom-onu-card`, `mushroom-isp-card`, `mushroom-omada-device(s)-card`, `mushroom-omada-clients-card`).

## Options
| Option | Meaning |
| --- | --- |
| `omada_host` | Controller URL, e.g. `https://192.168.0.2:443` |
| `omada_site` | Site name to monitor |
| `omada_insecure` | Skip TLS verification (self-signed controller cert) |
| `client_retention_hours` | Remove entities/trackers of clients not seen for this many hours (0 = never) |
| `speedtest_interval_minutes` | Run an Ookla speed test through every configured WAN this often (0 = off) |
| `speedtest_sources` | List of `{wan, ip}`: WAN name as Omada shows it, and a free LAN address the test binds to |
| `mqtt_interval` / `mqtt_expire_after` | Publish interval / entity expiry, seconds |
| `track_*` | Enable port, client, network-quality, DPI insight metrics |
| `mqtt_host` … | Only to override the Supervisor-provided broker |

Port 9202 (Prometheus `/metrics`, `/healthz`) is optional; map it only if you still scrape with Prometheus.

## Per-WAN speed test
Omada's own speed test needs gateway firmware support and runs on physical WANs only, so the add-on
measures from the LAN side instead. For every entry in `speedtest_sources` it adds the `ip` as an alias
on the host's primary interface (needs `host_network` and `NET_ADMIN`, both set) and runs
`speedtest -i <ip>` once per interval, one WAN after another. The gateway decides which ISP carries that
traffic, so on the Omada controller create, for each source address:

1. an IP group with the single address (`/32`), and
2. a **Policy Routing** rule: source = that IP group, destination = `IPGroup_Any`, interface = the WAN
   (or virtual WAN) to measure, placed **above** the catch-all rule.

Results appear as `net_wan_speedtest_*` entities on the WAN, with the server, ISP, result URL and the
last error as attributes. The throughput is capped by the add-on host's own link, so a 1 GbE host cannot
measure a 2.5 Gbit tariff.
