package collector

import (
	"log/slog"
	"time"

	"github.com/pzmonitor/internal/rcon"
	"github.com/prometheus/client_golang/prometheus"
)

// PZCollector implements prometheus.Collector. It queries RCON on each scrape.
type PZCollector struct {
	client *rcon.Client

	// Server health
	serverUp        *prometheus.Desc
	memoryUsed      *prometheus.Desc
	memoryTotal     *prometheus.Desc
	memoryMax       *prometheus.Desc
	fps             *prometheus.Desc
	avgUpdatePeriod *prometheus.Desc

	// Players & game world
	playersOnline    *prometheus.Desc
	playerOnline     *prometheus.Desc
	zombiesLoaded    *prometheus.Desc
	zombiesSimulated *prometheus.Desc
	zombiesTotal     *prometheus.Desc
	loadedCells      *prometheus.Desc
	animalsInstances *prometheus.Desc

	// Events (connection stats)
	zombiesKilledToday        *prometheus.Desc
	playersKilledByZombieToday *prometheus.Desc
	playersKilledByPlayerToday *prometheus.Desc
	playersKilledByFireToday   *prometheus.Desc
	zombifiedPlayersToday      *prometheus.Desc
	burnedCorpsesToday         *prometheus.Desc

	// Network
	networkSentBPS              *prometheus.Desc
	networkReceivedBPS          *prometheus.Desc
	networkSentBytes            *prometheus.Desc
	networkReceivedBytes        *prometheus.Desc
	networkLastActualBytesSent  *prometheus.Desc
	networkLastActualBytesRecv  *prometheus.Desc
	networkPacketLossTotal      *prometheus.Desc

	// Operational
	scrapeDuration *prometheus.Desc
}

func newDesc(name, help string) *prometheus.Desc {
	return prometheus.NewDesc(name, help, nil, nil)
}

// New creates a new PZCollector.
func New(client *rcon.Client) *PZCollector {
	return &PZCollector{
		client: client,

		serverUp:        newDesc("pz_server_up", "1 if RCON reachable, 0 otherwise"),
		memoryUsed:      newDesc("pz_memory_used_megabytes", "JVM memory used (MB)"),
		memoryTotal:     newDesc("pz_memory_total_megabytes", "JVM memory total (MB)"),
		memoryMax:       newDesc("pz_memory_max_megabytes", "JVM memory max (MB)"),
		fps:             newDesc("pz_fps", "Server FPS (tick rate)"),
		avgUpdatePeriod: newDesc("pz_avg_update_period_ms", "Average update period"),

		playersOnline:    newDesc("pz_players_online", "Current connected player count"),
		playerOnline:     prometheus.NewDesc("pz_player_online", "1 if player is currently connected", []string{"name"}, nil),
		zombiesLoaded:    newDesc("pz_zombies_loaded", "Currently loaded zombies"),
		zombiesSimulated: newDesc("pz_zombies_simulated", "Currently simulated zombies"),
		zombiesTotal:     newDesc("pz_zombies_total", "Total zombies in world"),
		loadedCells:      newDesc("pz_loaded_cells", "Number of loaded map cells"),
		animalsInstances: newDesc("pz_animals_instances", "Animal instances"),

		zombiesKilledToday:         newDesc("pz_zombies_killed_today", "Zombies killed today"),
		playersKilledByZombieToday: newDesc("pz_players_killed_by_zombie_today", "Players killed by zombies today"),
		playersKilledByPlayerToday: newDesc("pz_players_killed_by_player_today", "Players killed by players today"),
		playersKilledByFireToday:   newDesc("pz_players_killed_by_fire_today", "Players killed by fire today"),
		zombifiedPlayersToday:      newDesc("pz_zombified_players_today", "Zombified players today"),
		burnedCorpsesToday:         newDesc("pz_burned_corpses_today", "Burned corpses today"),

		networkSentBPS:         newDesc("pz_network_sent_bps", "Bytes per second sent"),
		networkReceivedBPS:     newDesc("pz_network_received_bps", "Bytes per second received"),
		networkSentBytes:            newDesc("pz_network_sent_bytes", "Total bytes sent"),
		networkReceivedBytes:        newDesc("pz_network_received_bytes", "Total bytes received"),
		networkLastActualBytesSent:  newDesc("pz_network_last_actual_bytes_sent", "Last actual bytes sent"),
		networkLastActualBytesRecv:  newDesc("pz_network_last_actual_bytes_received", "Last actual bytes received"),
		networkPacketLossTotal:      newDesc("pz_network_packet_loss_total", "Total packet loss"),

		scrapeDuration: newDesc("pz_scrape_duration_seconds", "Time taken for the RCON scrape"),
	}
}

func (c *PZCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.serverUp
	ch <- c.memoryUsed
	ch <- c.memoryTotal
	ch <- c.memoryMax
	ch <- c.fps
	ch <- c.avgUpdatePeriod
	ch <- c.playersOnline
	ch <- c.playerOnline
	ch <- c.zombiesLoaded
	ch <- c.zombiesSimulated
	ch <- c.zombiesTotal
	ch <- c.loadedCells
	ch <- c.animalsInstances
	ch <- c.zombiesKilledToday
	ch <- c.playersKilledByZombieToday
	ch <- c.playersKilledByPlayerToday
	ch <- c.playersKilledByFireToday
	ch <- c.zombifiedPlayersToday
	ch <- c.burnedCorpsesToday
	ch <- c.networkSentBPS
	ch <- c.networkReceivedBPS
	ch <- c.networkSentBytes
	ch <- c.networkReceivedBytes
	ch <- c.networkLastActualBytesSent
	ch <- c.networkLastActualBytesRecv
	ch <- c.networkPacketLossTotal
	ch <- c.scrapeDuration
}

func (c *PZCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()

	data, err := c.client.QueryAll()

	duration := time.Since(start).Seconds()
	ch <- prometheus.MustNewConstMetric(c.scrapeDuration, prometheus.GaugeValue, duration)

	if err != nil {
		slog.Error("rcon scrape failed", "error", err)
		ch <- prometheus.MustNewConstMetric(c.serverUp, prometheus.GaugeValue, 0)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.serverUp, prometheus.GaugeValue, 1)

	// Performance
	ch <- prometheus.MustNewConstMetric(c.memoryUsed, prometheus.GaugeValue, data.Performance["memory-used"])
	ch <- prometheus.MustNewConstMetric(c.memoryTotal, prometheus.GaugeValue, data.Performance["memory-total"])
	ch <- prometheus.MustNewConstMetric(c.memoryMax, prometheus.GaugeValue, data.Performance["memory-max"])
	ch <- prometheus.MustNewConstMetric(c.fps, prometheus.GaugeValue, data.Performance["fps"])
	ch <- prometheus.MustNewConstMetric(c.avgUpdatePeriod, prometheus.GaugeValue, data.Performance["avg-update-period"])

	// Players & game
	ch <- prometheus.MustNewConstMetric(c.playersOnline, prometheus.GaugeValue, float64(data.PlayerCount))
	for _, name := range data.PlayerNames {
		ch <- prometheus.MustNewConstMetric(c.playerOnline, prometheus.GaugeValue, 1, name)
	}
	ch <- prometheus.MustNewConstMetric(c.zombiesLoaded, prometheus.GaugeValue, data.Game["zombies-loaded"])
	ch <- prometheus.MustNewConstMetric(c.zombiesSimulated, prometheus.GaugeValue, data.Game["zombies-simulated"])
	ch <- prometheus.MustNewConstMetric(c.zombiesTotal, prometheus.GaugeValue, data.Game["zombies-total"])
	ch <- prometheus.MustNewConstMetric(c.loadedCells, prometheus.GaugeValue, data.Game["loaded-cells"])
	ch <- prometheus.MustNewConstMetric(c.animalsInstances, prometheus.GaugeValue, data.Game["animals-instances"])

	// Connection events
	ch <- prometheus.MustNewConstMetric(c.zombiesKilledToday, prometheus.GaugeValue, data.Connection["zombies-killed-today"])
	ch <- prometheus.MustNewConstMetric(c.playersKilledByZombieToday, prometheus.GaugeValue, data.Connection["players-killed-by-zombie-today"])
	ch <- prometheus.MustNewConstMetric(c.playersKilledByPlayerToday, prometheus.GaugeValue, data.Connection["players-killed-by-player-today"])
	ch <- prometheus.MustNewConstMetric(c.playersKilledByFireToday, prometheus.GaugeValue, data.Connection["players-killed-by-fire-today"])
	ch <- prometheus.MustNewConstMetric(c.zombifiedPlayersToday, prometheus.GaugeValue, data.Connection["zombified-players-today"])
	ch <- prometheus.MustNewConstMetric(c.burnedCorpsesToday, prometheus.GaugeValue, data.Connection["burned-corpses-today"])

	// Network
	ch <- prometheus.MustNewConstMetric(c.networkSentBPS, prometheus.GaugeValue, data.Network["sent-bps"])
	ch <- prometheus.MustNewConstMetric(c.networkReceivedBPS, prometheus.GaugeValue, data.Network["received-bps"])
	ch <- prometheus.MustNewConstMetric(c.networkSentBytes, prometheus.GaugeValue, data.Network["sent-bytes"])
	ch <- prometheus.MustNewConstMetric(c.networkReceivedBytes, prometheus.GaugeValue, data.Network["received-bytes"])
	ch <- prometheus.MustNewConstMetric(c.networkLastActualBytesSent, prometheus.GaugeValue, data.Network["last-actual-bytes-sent"])
	ch <- prometheus.MustNewConstMetric(c.networkLastActualBytesRecv, prometheus.GaugeValue, data.Network["last-actual-bytes-received"])
	ch <- prometheus.MustNewConstMetric(c.networkPacketLossTotal, prometheus.GaugeValue, data.Network["packet-loss-total"])
}
