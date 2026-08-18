# OpenVPN

[OpenVPN](https://openvpn.net) packaged as a [Syncloud](https://syncloud.org) app.

Gives you an encrypted tunnel back to your device from anywhere, and routes your
traffic through it.

## Layout

| Path | What |
|---|---|
| `cli/` | Go snap hooks (`install`, `configure`, `pre-refresh`, `post-refresh`) and the `bin/cli` Cobra binary |
| `backend/` | Go API on a unix socket: OIDC auth, `crypto/x509` PKI, OpenVPN management-interface client |
| `web/` | Vue 3 + Element Plus + Vite SPA, Playwright specs in `web/e2e/` |
| `openvpn/` | Upstream OpenVPN build, vendored with its shared libraries and loader |
| `nginx/` | Bundled nginx |
| `templates/` | `server.conf` / client `.ovpn` templates |
| `test/` | pytest device tests |

## Build

```
./cli/build.sh
./backend/build.sh
./web/build.sh
./nginx/build.sh
./openvpn/build.sh 2.7.6
./package.sh openvpn <build-number>
```

## Install on a device

```
snap install --devmode openvpn_<version>_<arch>.snap
```

## Upstream version

Pinned in `.drone.jsonnet` as `local openvpn = '...'`.
