# Changelog

All notable changes to this project are documented here.
The format is based on Keep a Changelog, and this project adheres to
Semantic Versioning (https://semver.org).

<!-- new-release -->

## [Unreleased]

- Subscribe to multiple topics per connection, each with its own QoS
- Collapse the connection panel to the saved list, opening the editor only for New or edit
- Add a toggle to order topics alphabetically or by received order
- Restore the connection status after a reload instead of showing a stale "disconnected"
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
