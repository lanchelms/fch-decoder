![fch-decoder](images/fch-decoder-header.png)

# fch-decoder

Decode, inspect, edit, and export metrics from Valheim `.fch` character files.
Use it as a Go library, one-shot JSON dumper, character editor, or Prometheus
scrape target.

## Tools

`fchdump` decodes one `.fch` file to formatted JSON.

`fchedit` validates and edits character data, writing changes in place or to a
copy.

`fchprom` serves Prometheus metrics from a Valheim character directory.

Go programs can import `github.com/lanchelms/fch-decoder` for structured
decode and encode behavior.

## Installation

### Go

Install the library:

```sh
go get github.com/lanchelms/fch-decoder
```

Install the command-line tools:

```sh
go install github.com/lanchelms/fch-decoder/cmd/fchdump@latest
go install github.com/lanchelms/fch-decoder/cmd/fchedit@latest
go install github.com/lanchelms/fch-decoder/cmd/fchprom@latest
```

### Docker

Pull the published images:

```sh
docker pull ghcr.io/lanchelms/fch-decoder-fchdump:latest
docker pull ghcr.io/lanchelms/fch-decoder-fchedit:latest
docker pull ghcr.io/lanchelms/fch-decoder-fchprom:latest
```

Or run them directly:

```sh
docker run --rm -v "$PWD/testdata:/data:ro" \
  ghcr.io/lanchelms/fch-decoder-fchdump:latest \
  --character /data/Steam_222222_bortson.fch
```

```sh
docker run --rm -p 9108:9108 \
  -v "$HOME/.config/unity3d/IronGate/Valheim/characters_local:/characters:ro" \
  ghcr.io/lanchelms/fch-decoder-fchprom:latest \
  --dir /characters --addr :9108
```

The bundled `docker-compose.yml` is for `fchprom`. Configure it with
`FCHPROM_CHARACTERS_DIR` and `FCHPROM_PORT`.

## fchdump

`fchdump` requires `--character` or the `CHARACTER` environment variable and
writes formatted JSON to stdout.

```sh
fchdump --character testdata/Steam_222222_bortson.fch
```

## fchedit

`fchedit` accepts these global flags:

```text
--character STRING   Character file to edit, also read from CHARACTER.
--out STRING         Write to this path instead of updating the input file.
--dry-run            Validate and summarize the edit without writing.
--no-backup          Do not create a backup before editing in place.
```

Commands:

```text
set skill <skill> <level>
set enemy <name> <value>
set material <name> <value>
set player-stat <stat> <value>
add inventory <item>
remove inventory <name>
list skills
list player-stats
list items
list inventory
```

Examples:

```sh
fchedit --character character.fch set skill Run 50
fchedit --character character.fch --dry-run add inventory 'Wood,stack=50,quality=1'
fchedit --character character.fch --out edited.fch set player-stat Deaths 0
```

## fchprom

`fchprom` serves metrics from a Valheim `characters_local` directory.

```text
--dir STRING              Valheim characters_local directory.
--addr STRING             Address to serve Prometheus metrics on. Default: :9108.
--metrics-path STRING     Prometheus metrics path. Default: /metrics.
--workers INT             Maximum files to decode in parallel. Default: 16.
--cache-ttl DURATION      How long to reuse decoded metrics. Default: 5s.
```

Local example:

```sh
fchprom --dir "$HOME/.config/unity3d/IronGate/Valheim/characters_local" --addr :9108
```

Compose example:

```sh
FCHPROM_CHARACTERS_DIR="$HOME/.config/unity3d/IronGate/Valheim/characters_local" \
FCHPROM_PORT=9108 \
docker compose up fchprom
```

## Go Library

Use the Go package when you want structured character data inside your own
application.

```go
package main

import (
	"fmt"
	"os"

	fch "github.com/lanchelms/fch-decoder"
)

func main() {
	file, err := os.Open("character.fch")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	character, err := fch.Decode(file)
	if err != nil {
		panic(err)
	}

	fmt.Println(character.Player.Name)
}
```
