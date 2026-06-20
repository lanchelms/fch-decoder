# fch-decoder

Go decoder for current Valheim `.fch` character files.

The decoder currently reads:

- file/version header and the 72-byte footer/hash block
- the embedded gzip map block metadata
- player name and player ID
- per-world key/value entries
- known text, enemy, material, and recipe stat arrays
- equipped guardian power block
- inventory item records

Tail sections after inventory are kept as `RemainingBytes` for now. In the sample files those bytes contain additional string-list sections such as known recipes/build pieces, tutorials, appearance, food, pins, and plugin custom data.

## CLI

```sh
go run ./cmd/fchdump 'testdata/Steam_76561198018104185_bortson.fch'
```

## Library

```go
f, err := os.Open("character.fch")
if err != nil {
	panic(err)
}
defer f.Close()

character, err := fch.Decode(f)
if err != nil {
	panic(err)
}
fmt.Println(character.Player.Name)
```

## Tests

```sh
go test ./...
```
