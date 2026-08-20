package playersdb

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestParseFixture(t *testing.T) {
	blob, err := os.ReadFile("testdata/admin.bin")
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(blob)
	if err != nil {
		t.Fatal(err)
	}
	if c.Forename != "Darrin" || c.Surname != "Lamb" {
		t.Errorf("name = %q %q", c.Forename, c.Surname)
	}
	if c.Profession != "base:constructionworker" {
		t.Errorf("profession = %q", c.Profession)
	}
	if int(c.X) != 1936 || int(c.Y) != 14413 {
		t.Errorf("pos = %v,%v", c.X, c.Y)
	}
	if c.ZombieKills != 8 || c.SurvivorKills != 0 {
		t.Errorf("kills = %d/%d", c.ZombieKills, c.SurvivorKills)
	}
	if c.HoursSurvived < 2.4 || c.HoursSurvived > 2.6 {
		t.Errorf("hours = %v", c.HoursSurvived)
	}
	if c.PerkLevels["Strength"] != 7 || c.PerkLevels["Fitness"] != 5 {
		t.Errorf("perks = %v", c.PerkLevels)
	}
	if len(c.BodyPartHealth) != bodyParts || c.Health() < 99 {
		t.Errorf("health = %v", c.BodyPartHealth)
	}
	if c.Items != 12 || len(c.Traits) != 5 {
		t.Errorf("items=%d traits=%d", c.Items, len(c.Traits))
	}
}

func TestParseTruncated(t *testing.T) {
	blob, _ := os.ReadFile("testdata/admin.bin")
	for _, n := range []int{0, 10, 100, 1000, len(blob) / 2} {
		if _, err := Parse(blob[:n]); err == nil {
			t.Errorf("expected error for %d bytes", n)
		}
	}
}

// TestLiveDB parses every row of a real players.db when PZ_PLAYERS_DB is set.
func TestLiveDB(t *testing.T) {
	path := os.Getenv("PZ_PLAYERS_DB")
	if path == "" {
		t.Skip("PZ_PLAYERS_DB not set")
	}
	rows, err := Open(path).Query(context.Background(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ParseErr != nil {
			t.Errorf("%s: %v", r.Label(), r.ParseErr)
			continue
		}
		c := r.Character
		if int(c.X) != int(r.X) || int(c.Y) != int(r.Y) {
			t.Errorf("%s: blob pos %v,%v != row %v,%v", r.Label(), c.X, c.Y, r.X, r.Y)
		}
		if c.Forename+" "+c.Surname != r.Name {
			t.Errorf("%s: blob name %q != row %q", r.Label(), c.Forename+" "+c.Surname, r.Name)
		}
		t.Logf("%s hours=%.1f zk=%d sk=%d dead=%v health=%.0f", r.Label(), c.HoursSurvived, c.ZombieKills, c.SurvivorKills, r.Dead, c.Health())
	}
}
