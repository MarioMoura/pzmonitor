package rcon

import (
	"testing"
)

func TestParsePlayersResponse(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantCount     int
		wantPlayers   []string
	}{
		{
			name:        "single player",
			input:       "Players connected (1):\n-ploita",
			wantCount:   1,
			wantPlayers: []string{"ploita"},
		},
		{
			name:        "multiple players",
			input:       "Players connected (3):\n-ploita\n-alice\n-bob",
			wantCount:   3,
			wantPlayers: []string{"ploita", "alice", "bob"},
		},
		{
			name:        "no players",
			input:       "Players connected (0):",
			wantCount:   0,
			wantPlayers: nil,
		},
		{
			name:        "empty response",
			input:       "",
			wantCount:   0,
			wantPlayers: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, players := ParsePlayersResponse(tt.input)
			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
			if len(players) != len(tt.wantPlayers) {
				t.Errorf("players = %v, want %v", players, tt.wantPlayers)
				return
			}
			for i, p := range players {
				if p != tt.wantPlayers[i] {
					t.Errorf("players[%d] = %q, want %q", i, p, tt.wantPlayers[i])
				}
			}
		})
	}
}

func TestParseStatsResponse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]float64
	}{
		{
			name: "performance stats",
			input: `memory-total: 4034.0
memory-free: 1584.0
memory-used: 2450.0
memory-max: 17179.0
fps: 104.0
max-update-period: 104.0
avg-update-period: 5.0
min-update-period: 99.0`,
			want: map[string]float64{
				"memory-total":      4034.0,
				"memory-free":       1584.0,
				"memory-used":       2450.0,
				"memory-max":        17179.0,
				"fps":               104.0,
				"max-update-period": 104.0,
				"avg-update-period": 5.0,
				"min-update-period": 99.0,
			},
		},
		{
			name: "game stats",
			input: `zombies-loaded: 0.0
zombies-culled: 0.0
animals-objects: 0.0
loaded-cells: 0.0
players: 0.0
zombies-simulated: 0.0
animals-instances: 0.0
zombies-teleports: 0.0
players-teleports: 0.0
zombies-updates: 0.0
zombies-updated: 0.0
zombies-total: 0.0`,
			want: map[string]float64{
				"zombies-loaded":     0.0,
				"zombies-culled":     0.0,
				"animals-objects":    0.0,
				"loaded-cells":       0.0,
				"players":            0.0,
				"zombies-simulated":  0.0,
				"animals-instances":  0.0,
				"zombies-teleports":  0.0,
				"players-teleports":  0.0,
				"zombies-updates":    0.0,
				"zombies-updated":    0.0,
				"zombies-total":      0.0,
			},
		},
		{
			name:  "empty response",
			input: "",
			want:  map[string]float64{},
		},
		{
			name:  "malformed lines ignored",
			input: "no-colon-here\nvalid-key: 42.0\njust text",
			want: map[string]float64{
				"valid-key": 42.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStatsResponse(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("got %d entries, want %d\ngot: %v", len(got), len(tt.want), got)
				return
			}
			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("key %q = %f, want %f", k, gotV, wantV)
				}
			}
		})
	}
}
