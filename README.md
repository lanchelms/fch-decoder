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
```

Or build the container image from the repository root:

```sh
docker build -f cmd/fchdump/Dockerfile -t fchdump .
```

### Prometheus Exporter

```sh
go install github.com/lanchelms/fch-decoder/cmd/fchprom@latest
```

Or build the container image from the repository root:

```sh
docker build -f cmd/fchprom/Dockerfile -t fchprom .
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

## CLI

`fchdump` decodes one character file and writes formatted JSON to stdout.

```sh
fchdump 'testdata/Steam_222222_bortson.fch'
```

Container example:

```sh
docker run --rm -v "$PWD/testdata:/data:ro" fchdump /data/Steam_222222_bortson.fch
```

## Prometheus Exporter

`fchprom` scans a Valheim `characters_local` directory and serves character metrics at `/metrics`.

```sh
fchprom -dir "$HOME/.config/unity3d/IronGate/Valheim/characters_local" -addr :9108
```

Container example:

```sh
docker run --rm -p 9108:9108 \
  -v "$HOME/.config/unity3d/IronGate/Valheim/characters_local:/characters:ro" \
  fchprom -dir /characters -addr :9108
```

The exporter reports skills, recipe counters, enemy counters, selected player stats, inferred distance metrics, and scrape errors. It ignores `.fch.old` and `backup_auto-*.fch` files.
