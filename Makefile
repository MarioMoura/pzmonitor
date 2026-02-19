.PHONY: build test clean run release

build:
	go build -o pzmonitor .

test:
	go test ./...

run: build
	@PZMONITOR_RCON_HOST=$$(awk -F' = ' '/^host/{print $$2}' ~/.rcon.conf) \
	PZMONITOR_RCON_PORT=$$(awk -F' = ' '/^port/{print $$2}' ~/.rcon.conf) \
	PZMONITOR_RCON_PASSWORD=$$(awk -F' = ' '/^passwd/{print $$2}' ~/.rcon.conf) \
	./pzmonitor

clean:
	rm -f pzmonitor

release:
	GITHUB_TOKEN=$$(gh auth token) goreleaser release --clean
