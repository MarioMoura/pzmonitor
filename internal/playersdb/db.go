package playersdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "modernc.org/sqlite"
)

// Row is one networkPlayers record with its decoded character.
type Row struct {
	Username    string
	PlayerIndex int
	Name        string
	SteamID     string
	X, Y, Z     float64
	Dead        bool
	Character   *Character
	ParseErr    error
}

// Label returns the username, suffixed with the slot index for split-screen
// characters, matching how the server names them.
func (r Row) Label() string {
	if r.PlayerIndex > 0 {
		return fmt.Sprintf("%s/%d", r.Username, r.PlayerIndex)
	}
	return r.Username
}

// DB reads players.db. Each Query opens the file read-only and immutable so
// no locks are taken against the running server; a write in progress just
// fails that one read.
type DB struct {
	dsn string
}

func Open(path string) *DB {
	return &DB{dsn: "file:" + url.PathEscape(path) + "?mode=ro&immutable=1"}
}

func (d *DB) Query(ctx context.Context, timeout time.Duration) ([]Row, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	db, err := sql.Open("sqlite", d.dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx,
		"SELECT username, playerIndex, name, COALESCE(steamid, ''), x, y, z, isDead, data FROM networkPlayers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var r Row
		var blob []byte
		if err := rows.Scan(&r.Username, &r.PlayerIndex, &r.Name, &r.SteamID, &r.X, &r.Y, &r.Z, &r.Dead, &blob); err != nil {
			return nil, err
		}
		r.Character, r.ParseErr = Parse(blob)
		out = append(out, r)
	}
	return out, rows.Err()
}
