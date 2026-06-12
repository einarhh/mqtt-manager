# Changelog

All notable changes to this project are documented here.
The format is based on Keep a Changelog, and this project adheres to
Semantic Versioning (https://semver.org).

<!-- new-release -->

## [1.2.0] - 2026-06-12

- Support plugins
- Add UI screenshot to the README
- Add MIT license and open-source prep
- Re-sign the bundle after swapping the icns
- Add make dist target for a shareable universal macOS build

## [1.1.0] - 2026-06-11

- Use the app icon as the header logo
- Add native menu with an About item showing the version
- Generate complete multi-resolution macOS icns after build
- Document the Make build and release workflow in the README

## [1.0.0] - 2026-06-11

- Add semantic versioning and Make-driven release tooling
- Add app icon: topic-tree node graph on a blue squircle
- UI: password on its own line, click row to toggle, larger tree arrow
- Ingest each message batch exactly once
- Re-subscribe on reconnect so messages resume after sleep/wake
- Initial commit: MQTT Manager v1 (Wails + Go + Svelte)
