# MQTT Manager

A desktop MQTT client — a live topic tree, per-topic value inspection with history,
publishing, and saved broker connections. Built with [Wails](https://wails.io) (Go backend +
Svelte/TypeScript frontend).

## Features (v1)

- **Connections** — save multiple broker profiles (host/port, TLS, auth, client ID, default
  subscription). Stored at `<user-config-dir>/mqtt-manager/profiles.json`.
- **TLS** — connect over `ssl://` with an optional CA certificate and an insecure-skip-verify
  toggle.
- **Live topic tree** — auto-builds a hierarchical tree from incoming messages, with a filter
  box, value previews, message counts, and update flashes.
- **Topic detail** — latest payload (JSON pretty-printed when applicable), metadata
  (QoS / retained / timestamp / count), and a recent value history.
- **Publish** — send messages with topic, payload, QoS, and retain flag.

Incoming messages are throttled on the Go side (batched every 100 ms) so a busy broker can't
flood the UI.

## Architecture

```
main.go / app.go              Wails bootstrap + JS-bound methods
internal/mqtt/                paho 3.1.1 client wrapper + message batcher (Connector interface)
internal/profiles/            connection-profile storage + TLS config builder
frontend/src/                 Svelte UI (ConnectionPanel, TopicTree, TopicDetail, PublishPanel)
```

The MQTT layer sits behind a `Connector` interface so an MQTT 5 (autopaho) backend can be
added later without touching the app layer.

## Development

Requires Go, Node, and the Wails CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@latest`).

```sh
wails dev      # hot-reload dev mode
wails build    # production .app bundle in build/bin
```

## Verifying

Point a profile at a broker (e.g. a local `eclipse-mosquitto` container or
`test.mosquitto.org:1883`), connect, subscribe to `#`, and publish a message — it should
appear in the tree and detail panel.

## Notes / roadmap

- v1 stores passwords in plaintext on disk — OS keychain storage is planned.
- Roadmap: MQTT 5, numeric charts/sparklines, retained-message deletion, WebSocket transport,
  message search/export.
