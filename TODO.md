# TODO

- [ ] Dashboard: add an `instance` template variable and filter every query with `{instance=~"$instance"}` so multiple servers can share one dashboard
- [ ] Dashboard: "Survivors" table (hours survived, zombie kills, health, dead) from `pz_character_*`
- [ ] Document running one instance per server (distinct `PZMONITOR_LISTEN_ADDR` and Prometheus `instance` label per target)
