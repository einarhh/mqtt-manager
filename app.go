package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/menu"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"mqtt-manager/internal/mqtt"
	"mqtt-manager/internal/plugins"
	"mqtt-manager/internal/profiles"
)

// Wails event names emitted to the frontend.
const (
	eventMessages = "mqtt:messages"
	eventStatus   = "mqtt:status"
)

// App is the Wails application backend bound to the frontend.
type App struct {
	ctx     context.Context
	client  *mqtt.Client
	store   *profiles.Store
	plugins *plugins.Store
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// Version returns the application version (injected at build time).
func (a *App) Version() string {
	return version
}

// showAbout displays a native About dialog with the version. It is wired to the
// Help > About menu item.
func (a *App) showAbout(_ *menu.CallbackData) {
	wruntime.MessageDialog(a.ctx, wruntime.MessageDialogOptions{
		Type:    wruntime.InfoDialog,
		Title:   "About MQTT Manager",
		Message: fmt.Sprintf("MQTT Manager\nVersion %s\n\n© 2026 Einar Helseth", version),
	})
}

// startup is called when the app starts. It wires up the MQTT client and the
// profile store, forwarding message batches and status changes to the frontend.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	a.client = mqtt.NewClient(
		func(batch []mqtt.Message) {
			wruntime.EventsEmit(a.ctx, eventMessages, batch)
		},
		func(status, detail string) {
			wruntime.EventsEmit(a.ctx, eventStatus, map[string]string{
				"status": status,
				"detail": detail,
			})
		},
	)

	store, err := profiles.NewStore()
	if err != nil {
		wruntime.LogError(a.ctx, "profile store init: "+err.Error())
		return
	}
	a.store = store

	pluginStore, err := plugins.NewStore()
	if err != nil {
		wruntime.LogError(a.ctx, "plugin store init: "+err.Error())
		return
	}
	a.plugins = pluginStore
}

// shutdown tears down the MQTT connection cleanly.
func (a *App) shutdown(_ context.Context) {
	if a.client != nil {
		a.client.Close()
	}
}

// --- Connection -----------------------------------------------------------

// Connect opens a connection using the given profile.
func (a *App) Connect(p profiles.ConnectionProfile) error {
	return a.client.Connect(p)
}

// Disconnect closes the active connection.
func (a *App) Disconnect() {
	a.client.Disconnect()
}

// Subscribe subscribes to a topic filter.
func (a *App) Subscribe(filter string, qos int) error {
	return a.client.Subscribe(filter, byte(qos))
}

// Unsubscribe removes a subscription.
func (a *App) Unsubscribe(filter string) error {
	return a.client.Unsubscribe(filter)
}

// Publish publishes a message. payloadB64 is base64-encoded so binary payloads
// round-trip safely from the frontend.
func (a *App) Publish(topic, payloadB64 string, qos int, retain bool) error {
	payload, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		return err
	}
	return a.client.Publish(topic, payload, byte(qos), retain)
}

// --- Profiles -------------------------------------------------------------

// ListProfiles returns all saved connection profiles.
func (a *App) ListProfiles() ([]profiles.ConnectionProfile, error) {
	if a.store == nil {
		return []profiles.ConnectionProfile{}, nil
	}
	return a.store.List()
}

// SaveProfile inserts or updates a profile and returns the stored value.
func (a *App) SaveProfile(p profiles.ConnectionProfile) (profiles.ConnectionProfile, error) {
	return a.store.Save(p)
}

// DeleteProfile removes a profile by ID.
func (a *App) DeleteProfile(id string) error {
	return a.store.Delete(id)
}

// --- Plugins --------------------------------------------------------------

// ListPlugins returns all saved decoder plugins.
func (a *App) ListPlugins() ([]plugins.Plugin, error) {
	if a.plugins == nil {
		return []plugins.Plugin{}, nil
	}
	return a.plugins.List()
}

// SavePlugin inserts or updates a decoder plugin and returns the stored value.
func (a *App) SavePlugin(p plugins.Plugin) (plugins.Plugin, error) {
	return a.plugins.Save(p)
}

// DeletePlugin removes a decoder plugin by ID.
func (a *App) DeletePlugin(id string) error {
	return a.plugins.Delete(id)
}

var jsFilter = []wruntime.FileFilter{{DisplayName: "JavaScript (*.js)", Pattern: "*.js"}}

// ExportPlugin writes a plugin's source to a user-chosen .js file. A cancelled
// dialog is a no-op.
func (a *App) ExportPlugin(id string) error {
	if a.plugins == nil {
		return nil
	}
	list, err := a.plugins.List()
	if err != nil {
		return err
	}
	var p *plugins.Plugin
	for i := range list {
		if list[i].ID == id {
			p = &list[i]
			break
		}
	}
	if p == nil {
		return fmt.Errorf("plugin %s not found", id)
	}
	name := p.Filename
	if name == "" {
		name = "plugin.js"
	}
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Export plugin",
		DefaultFilename: name,
		Filters:         jsFilter,
	})
	if err != nil {
		return err
	}
	if path == "" {
		return nil // cancelled
	}
	return os.WriteFile(path, []byte(p.Source), 0o644)
}

// ImportPlugin opens a .js file and saves its contents as a new plugin, which
// it returns. A cancelled dialog returns a zero-value plugin (empty ID).
func (a *App) ImportPlugin() (plugins.Plugin, error) {
	if a.plugins == nil {
		return plugins.Plugin{}, nil
	}
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:   "Import plugin",
		Filters: jsFilter,
	})
	if err != nil {
		return plugins.Plugin{}, err
	}
	if path == "" {
		return plugins.Plugin{}, nil // cancelled
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return plugins.Plugin{}, fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if name == "" {
		name = "imported"
	}
	return a.plugins.Save(plugins.Plugin{
		Name:    name,
		Enabled: true,
		Source:  string(data),
	})
}
