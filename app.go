package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/menu"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"mqtt-manager/internal/mqtt"
	"mqtt-manager/internal/plugins"
	"mqtt-manager/internal/profiles"
)

// Wails event names emitted to the frontend. Payloads are tagged with the
// connection ID so the frontend can route them to the right connection.
const (
	eventMessages = "mqtt:messages"
	eventStatus   = "mqtt:status"
	eventPlugins  = "plugins:changed"
)

// ConnMessages is a batch of received messages tagged with its connection ID.
type ConnMessages struct {
	ID       string         `json:"id"`
	Messages []mqtt.Message `json:"messages"`
}

// ConnState is a connection's current status, tagged with its ID. It is both the
// status-event payload and the element type returned by Connections().
type ConnState struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// App is the Wails application backend bound to the frontend.
type App struct {
	ctx          context.Context
	mu           sync.Mutex
	clients      map[string]*mqtt.Client // keyed by profile ID (one connection per profile)
	store        *profiles.Store
	plugins      *plugins.Store
	pluginsWatch func() // stops the plugin-directory watcher
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{clients: map[string]*mqtt.Client{}}
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

// startup is called when the app starts. It wires up the profile and plugin
// stores. MQTT clients are created lazily, one per profile, on Connect.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

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

	// Watch the plugins directory so edits made outside the app auto-reload.
	stop, err := pluginStore.Watch(func() {
		wruntime.EventsEmit(a.ctx, eventPlugins)
	})
	if err != nil {
		wruntime.LogError(a.ctx, "plugin watcher: "+err.Error())
	} else {
		a.pluginsWatch = stop
	}
}

// shutdown tears down all MQTT connections cleanly.
func (a *App) shutdown(_ context.Context) {
	if a.pluginsWatch != nil {
		a.pluginsWatch()
	}

	a.mu.Lock()
	clients := make([]*mqtt.Client, 0, len(a.clients))
	for _, c := range a.clients {
		clients = append(clients, c)
	}
	a.clients = map[string]*mqtt.Client{}
	a.mu.Unlock()

	for _, c := range clients {
		c.Close()
	}
}

// --- Connection -----------------------------------------------------------

// client returns the client for a connection ID, or nil if none exists.
func (a *App) client(id string) *mqtt.Client {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.clients[id]
}

// Connect opens a connection using the given profile. The connection ID is the
// profile ID, so connecting an already-connected profile reuses its client.
func (a *App) Connect(p profiles.ConnectionProfile) error {
	id := p.ID
	a.mu.Lock()
	c := a.clients[id]
	if c == nil {
		// Callbacks close over the id so every emit is tagged with its connection.
		c = mqtt.NewClient(
			func(batch []mqtt.Message) {
				wruntime.EventsEmit(a.ctx, eventMessages, ConnMessages{ID: id, Messages: batch})
			},
			func(status, detail string) {
				wruntime.EventsEmit(a.ctx, eventStatus, ConnState{ID: id, Status: status, Detail: detail})
			},
		)
		a.clients[id] = c
	}
	a.mu.Unlock()
	return c.Connect(p)
}

// Disconnect closes the connection's broker link but keeps its client (and the
// frontend's captured tree) until RemoveConnection is called.
func (a *App) Disconnect(id string) {
	if c := a.client(id); c != nil {
		c.Disconnect()
	}
}

// RemoveConnection disconnects, tears down the client (stopping its batcher), and
// forgets it entirely.
func (a *App) RemoveConnection(id string) {
	a.mu.Lock()
	c := a.clients[id]
	delete(a.clients, id)
	a.mu.Unlock()
	if c != nil {
		c.Close()
	}
}

// Connections returns the current status of every live connection. The frontend
// calls this on load to rebuild its connection list after a reload, since status
// is otherwise only pushed on transitions.
func (a *App) Connections() []ConnState {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]ConnState, 0, len(a.clients))
	for id, c := range a.clients {
		status, detail := c.Status()
		out = append(out, ConnState{ID: id, Status: status, Detail: detail})
	}
	return out
}

// Subscribe subscribes a connection to a topic filter.
func (a *App) Subscribe(id, filter string, qos int) error {
	c := a.client(id)
	if c == nil {
		return fmt.Errorf("connection %s not found", id)
	}
	return c.Subscribe(filter, byte(qos))
}

// Unsubscribe removes a subscription from a connection.
func (a *App) Unsubscribe(id, filter string) error {
	c := a.client(id)
	if c == nil {
		return fmt.Errorf("connection %s not found", id)
	}
	return c.Unsubscribe(filter)
}

// Publish publishes a message on a connection. payloadB64 is base64-encoded so
// binary payloads round-trip safely from the frontend.
func (a *App) Publish(id, topic, payloadB64 string, qos int, retain bool) error {
	c := a.client(id)
	if c == nil {
		return fmt.Errorf("connection %s not found", id)
	}
	payload, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		return err
	}
	return c.Publish(topic, payload, byte(qos), retain)
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
