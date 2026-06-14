package profiles

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Subscription is a single topic filter and the QoS to subscribe it at.
type Subscription struct {
	Filter string `json:"filter"`
	QoS    byte   `json:"qos"` // 0,1,2
}

// ConnectionProfile describes a saved MQTT broker connection.
type ConnectionProfile struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Host          string         `json:"host"`
	Port          int            `json:"port"`
	UseTLS        bool           `json:"useTls"`
	TLSInsecure   bool           `json:"tlsInsecure"`
	CACertPath    string         `json:"caCertPath"`
	ClientID      string         `json:"clientId"`
	Username      string         `json:"username"`
	Password      string         `json:"password"`
	KeepAlive     int            `json:"keepAlive"`     // seconds; 0 -> default 30
	Subscriptions []Subscription `json:"subscriptions"` // topic filters to subscribe on connect

	// Legacy single-topic fields, kept for migrating older profiles.json files.
	// Normalize folds these into Subscriptions and clears them.
	SubFilter string `json:"subFilter,omitempty"`
	SubQoS    byte   `json:"subQos,omitempty"`
}

// Normalize migrates the legacy single-topic fields into Subscriptions so older
// saved profiles keep working. It is idempotent.
func (p *ConnectionProfile) Normalize() {
	if len(p.Subscriptions) == 0 && p.SubFilter != "" {
		p.Subscriptions = []Subscription{{Filter: p.SubFilter, QoS: p.SubQoS}}
	}
	p.SubFilter = ""
	p.SubQoS = 0
}

// BrokerURL builds the scheme://host:port string paho expects.
func (p ConnectionProfile) BrokerURL() string {
	scheme := "tcp"
	if p.UseTLS {
		scheme = "ssl"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, p.Host, p.Port)
}

// KeepAliveSeconds returns the keep-alive, defaulting to 30s.
func (p ConnectionProfile) KeepAliveSeconds() int {
	if p.KeepAlive <= 0 {
		return 30
	}
	return p.KeepAlive
}

// TLSConfig builds a *tls.Config from the profile, or nil if TLS is disabled.
func (p ConnectionProfile) TLSConfig() (*tls.Config, error) {
	if !p.UseTLS {
		return nil, nil
	}
	cfg := &tls.Config{
		InsecureSkipVerify: p.TLSInsecure, //nolint:gosec // user-controlled toggle
		MinVersion:         tls.VersionTLS12,
	}
	if p.CACertPath != "" {
		pem, err := os.ReadFile(p.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in %s", p.CACertPath)
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}
