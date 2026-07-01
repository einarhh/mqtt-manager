# Changelog

All notable changes to this project are documented here.
The format is based on Keep a Changelog, and this project adheres to
Semantic Versioning (https://semver.org).

<!-- new-release -->

## [1.4.1] - 2026-07-01

- Add GitHub Actions release workflow for macOS and Windows builds

## [1.4.0] - 2026-06-23

- Move the Clear button away from the filter and give it a trash icon so it's no longer mistaken for clearing the filter
- Pulse a small dot on topics (and their parents) when a new message arrives
- Add a Copy button to the detail panel for copying a topic's current value
- Show the date on history timestamps when a message isn't from today
- Stop the detail panel from twitching as messages stream in
- Show the active connection's name in the header and drop the duplicate counter
- Drag to resize the connection, topic, and detail columns (sizes are remembered)
- Let decoder plugins render their own HTML to take full control of the detail panel
- Add subtree decoder plugins that summarize a whole topic group into a single card
- Import and export decoder plugins as files from the plugin manager
- Reload plugins automatically when their files change on disk, with a manual Reload button too

## [1.3.0] - 2026-06-15

- Chart numeric topic values over time in the topic detail panel, MQTT-Explorer-style
- Connect to multiple brokers simultaneously and switch between them by selecting a connection in the list
- Subscribe to multiple topics per connection, each with its own QoS
- Collapse the connection panel to the saved list, opening the editor only for New or edit
- Add a toggle to order topics alphabetically or by received order
- Restore the connection status after a reload instead of showing a stale "disconnected"
- Make connecting clearer with an always-visible Connect button on disconnected connections, and widen the connection list
- Decode GPS coordinates from payloads
- Refresh the README screenshot to showcase plugin decoding

## [1.2.0] - 2026-06-12

- Add plugin support for decoding message payloads
- Open-source the project under the MIT license
- Ship a shareable universal macOS build via `make dist`
- Add a UI screenshot to the README

## [1.1.0] - 2026-06-11

- Add a native menu with an About item showing the version
- Use the app icon as the header logo
- Generate a complete multi-resolution macOS icon on build
- Document the Make build and release workflow in the README

## [1.0.0] - 2026-06-11

- Initial release: MQTT Manager (Wails + Go + Svelte)
- Browse topics in a live tree with per-topic message detail
- Re-subscribe automatically on reconnect after sleep/wake
- Add semantic versioning and Make-driven release tooling
- Add app icon: topic-tree node graph on a blue squircle
