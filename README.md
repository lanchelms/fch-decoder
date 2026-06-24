![fch-decoder](images/fch-decoder-header.png)

# fch-decoder

Decode current Valheim `.fch` character files from Go, the command line, or a Prometheus scrape target. It reads player identity, stats, inventory, foods, skills, unlocks, known lists, and other saved character data.

## How To Install

### Library

```sh
go get github.com/lanchelms/fch-decoder
```

### CLI

```sh
go install github.com/lanchelms/fch-decoder/cmd/fchdump@latest
go install github.com/lanchelms/fch-decoder/cmd/fchedit@latest
```

Or pull the published container image:

```sh
docker pull ghcr.io/lanchelms/fch-decoder-fchdump:latest
docker pull ghcr.io/lanchelms/fch-decoder-fchedit:latest
```

### Prometheus Exporter

```sh
go install github.com/lanchelms/fch-decoder/cmd/fchprom@latest
```

Or pull the published container image:

```sh
docker pull ghcr.io/lanchelms/fch-decoder-fchprom:latest
```

## Library

Use the Go package when you want structured character data inside your own application.

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

The decoded trailer includes `hashValid`, which verifies Valheim's current
SHA-512 trailer hash over the inner character payload bytes.

## CLI

`fchdump` decodes one character file and writes formatted JSON to stdout.

```sh
fchdump 'testdata/Steam_222222_bortson.fch'
```

Container example:

```sh
docker run --rm -v "$PWD/testdata:/data:ro" \
  ghcr.io/lanchelms/fch-decoder-fchdump:latest /data/Steam_222222_bortson.fch
```

`fchedit` decodes one character file, applies requested edits, recalculates
the payload length and trailer hash, and writes the edited file to `-out` unless
`-in-place` is specified.

```sh
fchedit -out edited.fch \
  -set-player-stat Deaths=0 \
  -set-skill-level Run=50 \
  -set-enemy-stat '$enemy_greydwarf=25' \
  -set-material-stat '$item_wood=100' \
  -add-inventory 'Wood,stack=50,quality=1' \
  -remove-inventory Stone \
  character.fch
```

Each edit flag may be repeated. Inventory additions accept
`name[,stack=n,durability=n,grid-x=n,grid-y=n,equipped=bool,quality=n,variant=n,crafter-id=n,crafter-name=s,world-level=n,picked-up=bool]`.

## Prometheus Exporter

`fchprom` scans a Valheim `characters_local` directory and serves character metrics at `/metrics`.

```sh
fchprom -dir "$HOME/.config/unity3d/IronGate/Valheim/characters_local" -addr :9108
```

Container example:

```sh
docker run --rm -p 9108:9108 \
  -v "$HOME/.config/unity3d/IronGate/Valheim/characters_local:/characters:ro" \
  ghcr.io/lanchelms/fch-decoder-fchprom:latest -dir /characters -addr :9108
```

Example metric series:

```prometheus
valheim_character_skills{player="Bortson",skill="Run"}
valheim_character_crafting{player="Bortson",recipe="AxeStone"}
valheim_character_enemies{player="Bortson",enemy="Greyling"}
valheim_character_stats{player="Bortson",stat="ArrowsShot"}
valheim_character_distance{player="Bortson",mode="Total"}
valheim_character_scrape_errors
```

The exporter ignores `.fch.old` and `backup_auto-*.fch` files.
