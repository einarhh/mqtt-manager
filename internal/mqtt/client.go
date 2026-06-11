package mqtt

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"mqtt-manager/internal/profiles"
)

// Connection status values reported to the frontend.
const (
	StatusDisconnected = "disconnected"
	StatusConnecting   = "connecting"
	StatusConnected    = "connected"
	StatusReconnecting = "reconnecting"
	StatusError        = "error"
)

// Connector is the abstraction over an MQTT backend. paho 3.1.1 is the v1
// implementation; an MQTT 5 (autopaho) backend can satisfy the same interface
// later without touching the app layer.
type Connector interface {
	Connect(p profiles.ConnectionProfile) error
	Disconnect()
	Publish(topic string, payload []byte, qos byte, retain bool) error
	Subscribe(filter string, qos byte) error
	Unsubscribe(filter string) error
}

// StatusFunc reports connection state changes (status, human-readable detail).
type StatusFunc func(status, detail string)

// Client wraps a paho MQTT client and pushes received messages through a
// batcher to the supplied flush callback.
type Client struct {
	mu       sync.Mutex
	client   paho.Client
	batcher  *batcher
	onStatus StatusFunc
	subs     map[string]byte // active subscriptions (filter -> qos), for resubscribe
}

// NewClient builds a client. onMessages receives throttled batches of incoming
// messages; onStatus receives connection-state transitions.
func NewClient(onMessages func([]Message), onStatus StatusFunc) *Client {
	return &Client{
		batcher:  newBatcher(100*time.Millisecond, onMessages),
		onStatus: onStatus,
		subs:     map[string]byte{},
	}
}

var _ Connector = (*Client)(nil)

func (c *Client) Connect(p profiles.ConnectionProfile) error {
	c.Disconnect() // ensure any prior connection is torn down

	tlsCfg, err := p.TLSConfig()
	if err != nil {
		return err
	}

	opts := paho.NewClientOptions().
		AddBroker(p.BrokerURL()).
		SetClientID(p.ClientID).
		SetKeepAlive(time.Duration(p.KeepAliveSeconds()) * time.Second).
		SetAutoReconnect(true).
		SetConnectRetry(false).
		SetCleanSession(true).
		SetOrderMatters(false)

	if p.Username != "" {
		opts.SetUsername(p.Username)
	}
	if p.Password != "" {
		opts.SetPassword(p.Password)
	}
	if tlsCfg != nil {
		opts.SetTLSConfig(tlsCfg)
	}

	opts.SetDefaultPublishHandler(c.handleMessage)
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		c.status(StatusReconnecting, err.Error())
	})
	opts.SetOnConnectHandler(func(_ paho.Client) {
		c.status(StatusConnected, p.BrokerURL())
		// Fires on the initial connect AND every auto-reconnect. Because we use a
		// clean session, the broker keeps no subscription state across a reconnect
		// (e.g. after the machine sleeps), so we must restore them ourselves.
		go c.resubscribe()
	})

	c.status(StatusConnecting, p.BrokerURL())

	client := paho.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(15 * time.Second) {
		c.status(StatusError, "connection timed out")
		return fmt.Errorf("connection to %s timed out", p.BrokerURL())
	}
	if err := token.Error(); err != nil {
		c.status(StatusError, err.Error())
		return err
	}

	c.mu.Lock()
	c.client = client
	c.mu.Unlock()
	return nil
}

func (c *Client) Disconnect() {
	c.mu.Lock()
	client := c.client
	c.client = nil
	c.mu.Unlock()

	if client != nil && client.IsConnectionOpen() {
		client.Disconnect(250)
	}
	if client != nil {
		c.status(StatusDisconnected, "")
	}

	// Explicit disconnect clears the subscription set so a later Connect starts clean.
	c.mu.Lock()
	c.subs = map[string]byte{}
	c.mu.Unlock()
}

func (c *Client) Subscribe(filter string, qos byte) error {
	client := c.current()
	if client == nil {
		return fmt.Errorf("not connected")
	}
	token := client.Subscribe(filter, qos, nil)
	token.Wait()
	if err := token.Error(); err != nil {
		return err
	}
	// Remember it so we can restore the subscription after an auto-reconnect.
	c.mu.Lock()
	c.subs[filter] = qos
	c.mu.Unlock()
	return nil
}

func (c *Client) Unsubscribe(filter string) error {
	client := c.current()
	if client == nil {
		return fmt.Errorf("not connected")
	}
	token := client.Unsubscribe(filter)
	token.Wait()
	if err := token.Error(); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.subs, filter)
	c.mu.Unlock()
	return nil
}

// resubscribe re-establishes all tracked subscriptions on the current client.
// Called from the OnConnect handler after every (re)connect.
func (c *Client) resubscribe() {
	c.mu.Lock()
	client := c.client
	subs := make(map[string]byte, len(c.subs))
	for f, q := range c.subs {
		subs[f] = q
	}
	c.mu.Unlock()

	if client == nil || len(subs) == 0 {
		return
	}
	for filter, qos := range subs {
		token := client.Subscribe(filter, qos, nil)
		token.Wait()
		if err := token.Error(); err != nil {
			c.status(StatusError, fmt.Sprintf("resubscribe %s: %s", filter, err))
		}
	}
}

func (c *Client) Publish(topic string, payload []byte, qos byte, retain bool) error {
	client := c.current()
	if client == nil {
		return fmt.Errorf("not connected")
	}
	token := client.Publish(topic, qos, retain, payload)
	token.Wait()
	return token.Error()
}

// Close tears down the connection and stops the batcher. Call on shutdown.
func (c *Client) Close() {
	c.Disconnect()
	c.batcher.stop()
}

func (c *Client) handleMessage(_ paho.Client, m paho.Message) {
	c.batcher.add(Message{
		Topic:     m.Topic(),
		Payload:   base64.StdEncoding.EncodeToString(m.Payload()),
		QoS:       m.Qos(),
		Retained:  m.Retained(),
		Timestamp: time.Now().UnixMilli(),
	})
}

func (c *Client) current() paho.Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client
}

func (c *Client) status(s, detail string) {
	if c.onStatus != nil {
		c.onStatus(s, detail)
	}
}
