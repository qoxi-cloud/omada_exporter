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
| `mqtt_interval` / `mqtt_expire_after` | Publish interval / entity expiry, seconds |
| `track_*` | Enable port, client, network-quality, DPI insight metrics |
| `mqtt_host` … | Only to override the Supervisor-provided broker |

Port 9202 (Prometheus `/metrics`, `/healthz`) is optional; map it only if you still scrape with Prometheus.
