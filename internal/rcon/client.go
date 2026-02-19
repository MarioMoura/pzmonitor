package rcon

import (
	"fmt"

	gorcon "github.com/gorcon/rcon"
)

// Client wraps an RCON connection to a Project Zomboid server.
type Client struct {
	addr     string
	password string
}

// NewClient creates a new RCON client configuration.
func NewClient(addr, password string) *Client {
	return &Client{addr: addr, password: password}
}

// QueryAll connects to the server, executes all stat commands, and returns parsed results.
// The connection is opened and closed within this call.
func (c *Client) QueryAll() (*ServerData, error) {
	conn, err := gorcon.Dial(c.addr, c.password)
	if err != nil {
		return nil, fmt.Errorf("rcon dial: %w", err)
	}
	defer conn.Close()

	data := &ServerData{}

	// Players
	resp, err := conn.Execute("players")
	if err != nil {
		return nil, fmt.Errorf("rcon execute players: %w", err)
	}
	data.PlayerCount, data.PlayerNames = ParsePlayersResponse(resp)

	// Stats categories
	categories := []struct {
		cmd  string
		dest *map[string]float64
	}{
		{"stats performance all", &data.Performance},
		{"stats game all", &data.Game},
		{"stats connection all", &data.Connection},
		{"stats network all", &data.Network},
	}

	for _, cat := range categories {
		resp, err := conn.Execute(cat.cmd)
		if err != nil {
			return nil, fmt.Errorf("rcon execute %q: %w", cat.cmd, err)
		}
		*cat.dest = ParseStatsResponse(resp)
	}

	return data, nil
}

// ServerData holds all parsed RCON data from a single scrape.
type ServerData struct {
	PlayerCount int
	PlayerNames []string
	Performance map[string]float64
	Game        map[string]float64
	Connection  map[string]float64
	Network     map[string]float64
}
