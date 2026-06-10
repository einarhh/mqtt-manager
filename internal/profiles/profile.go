package profiles

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// ConnectionProfile describes a saved MQTT broker connection.
type ConnectionProfile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	UseTLS      bool   `json:"useTls"`
	TLSInsecure bool   `json:"tlsInsecure"`
	CACertPath  string `json:"caCertPath"`
	ClientID    string `json:"clientId"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	KeepAlive   int    `json:"keepAlive"`   // seconds; 0 -> default 30
	SubFilter   string `json:"subFilter"`   // default subscribe topic filter
	SubQoS      byte   `json:"subQos"`      // 0,1,2
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
