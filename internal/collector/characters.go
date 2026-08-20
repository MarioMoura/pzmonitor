package collector

import (
	"context"
	"log/slog"
	"time"

	"github.com/MarioMoura/pzmonitor/internal/playersdb"
	"github.com/prometheus/client_golang/prometheus"
)

// CharacterCollector reads players.db on each scrape and exposes per-character
// metrics. It keeps no state between scrapes.
type CharacterCollector struct {
	db      *playersdb.DB
	timeout time.Duration

	up             *prometheus.Desc
	scrapeDuration *prometheus.Desc
	parseErrors    *prometheus.Desc
	info           *prometheus.Desc
	dead           *prometheus.Desc
	hoursSurvived  *prometheus.Desc
	zombieKills    *prometheus.Desc
	survivorKills  *prometheus.Desc
	health         *prometheus.Desc
	stat           *prometheus.Desc
	perkLevel      *prometheus.Desc
	position       *prometheus.Desc
	items          *prometheus.Desc
}

func NewCharacterCollector(db *playersdb.DB, timeout time.Duration) *CharacterCollector {
	l := []string{"username"}
	return &CharacterCollector{
		db:      db,
		timeout: timeout,

		up:             newDesc("pz_players_db_up", "1 if players.db was read successfully"),
		scrapeDuration: newDesc("pz_players_db_scrape_duration_seconds", "Time taken to read and parse players.db"),
		parseErrors:    newDesc("pz_players_db_parse_errors", "Characters whose save data could not be parsed in this scrape"),
		info:           prometheus.NewDesc("pz_character_info", "Character identity", []string{"username", "name", "profession", "steamid"}, nil),
		dead:           prometheus.NewDesc("pz_character_dead", "1 if the character is dead", l, nil),
		hoursSurvived:  prometheus.NewDesc("pz_character_hours_survived", "In-game hours survived", l, nil),
		zombieKills:    prometheus.NewDesc("pz_character_zombie_kills", "Zombies killed by the character", l, nil),
		survivorKills:  prometheus.NewDesc("pz_character_survivor_kills", "Survivors killed by the character", l, nil),
		health:         prometheus.NewDesc("pz_character_health", "Average body part health (0-100)", l, nil),
		stat:           prometheus.NewDesc("pz_character_stat", "Character stat (hunger, thirst, fatigue, ...)", []string{"username", "stat"}, nil),
		perkLevel:      prometheus.NewDesc("pz_character_perk_level", "Perk level", []string{"username", "perk"}, nil),
		position:       prometheus.NewDesc("pz_character_position", "Last saved world position", []string{"username", "axis"}, nil),
		items:          prometheus.NewDesc("pz_character_items", "Item stacks in the main inventory", l, nil),
	}
}

func (c *CharacterCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{c.up, c.scrapeDuration, c.parseErrors, c.info, c.dead, c.hoursSurvived,
		c.zombieKills, c.survivorKills, c.health, c.stat, c.perkLevel, c.position, c.items} {
		ch <- d
	}
}

func (c *CharacterCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	rows, err := c.db.Query(context.Background(), c.timeout)
	ch <- prometheus.MustNewConstMetric(c.scrapeDuration, prometheus.GaugeValue, time.Since(start).Seconds())
	if err != nil {
		slog.Error("players.db read failed", "error", err)
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

	parseErrors := 0
	gauge := func(d *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, labels...)
	}
	for _, r := range rows {
		u := r.Label()
		gauge(c.dead, b2f(r.Dead), u)
		gauge(c.position, r.X, u, "x")
		gauge(c.position, r.Y, u, "y")
		gauge(c.position, r.Z, u, "z")
		if r.ParseErr != nil {
			parseErrors++
			slog.Warn("character parse failed", "username", u, "error", r.ParseErr)
			gauge(c.info, 1, u, r.Name, "", r.SteamID)
			continue
		}
		p := r.Character
		gauge(c.info, 1, u, r.Name, p.Profession, r.SteamID)
		gauge(c.hoursSurvived, p.HoursSurvived, u)
		gauge(c.zombieKills, float64(p.ZombieKills), u)
		gauge(c.survivorKills, float64(p.SurvivorKills), u)
		gauge(c.health, p.Health(), u)
		gauge(c.items, float64(p.Items), u)
		for name, v := range p.Stats {
			gauge(c.stat, v, u, name)
		}
		for perk, lvl := range p.PerkLevels {
			gauge(c.perkLevel, float64(lvl), u, perk)
		}
	}
	gauge(c.parseErrors, float64(parseErrors))
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
