package main

import (
	"context"
	"encoding/base64"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"mqtt-manager/internal/mqtt"
	"mqtt-manager/internal/profiles"
)

// Wails event names emitted to the frontend.
const (
	eventMessages = "mqtt:messages"
	eventStatus   = "mqtt:status"
)

// App is the Wails application backend bound to the frontend.
type App struct {
	ctx    context.Context
	client *mqtt.Client
	store  *profiles.Store
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// Version returns the application version (injected at build time).
func (a *App) Version() string {
	return version
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
