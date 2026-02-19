package rcon

import (
	"strconv"
	"strings"
)

// ParsePlayersResponse parses the `players` RCON command response.
// Example: "Players connected (1):\n-ploita"
// Returns the player count and list of player names.
func ParsePlayersResponse(response string) (int, []string) {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	if len(lines) == 0 {
		return 0, nil
	}

	// Extract count from header: "Players connected (N):"
	header := lines[0]
	count := 0
	if start := strings.Index(header, "("); start != -1 {
		if end := strings.Index(header[start:], ")"); end != -1 {
			if n, err := strconv.Atoi(header[start+1 : start+end]); err == nil {
				count = n
			}
		}
	}

	var players []string
	for _, line := range lines[1:] {
		name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if name != "" {
			players = append(players, name)
		}
	}

	return count, players
}

// ParseStatsResponse parses a `stats <category> all` RCON command response.
// Each line has the format "key: value".
// Returns a map of key to float64 value.
func ParseStatsResponse(response string) map[string]float64 {
	result := make(map[string]float64)

	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])

		if val, err := strconv.ParseFloat(valStr, 64); err == nil {
			result[key] = val
		}
	}

	return result
}
