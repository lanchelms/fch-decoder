# fch-decoder

Go decoder for current Valheim `.fch` character files.

The decoder currently reads:

- file/version header and the 72-byte footer/hash block
- leading player stat float table
- the embedded gzip map block metadata
- player name and player ID
- per-world key/value entries
- known text, enemy, material, and recipe stat arrays
- known material and recipe lists
- player health/stamina state and equipped guardian power
- inventory item records
- player skill records

`materialStats` and `recipeStats` are the raw per-item stat counters saved in
the character file. They are not current inventory contents, and evidence from
stack-producing items suggests they should not be interpreted as lifetime item
quantities either. For example, crafted arrows and food can have stat values far
below the number of items produced because the game appears to count stat events
rather than stack amounts.

Most tail sections after inventory are consumed to reach the skill records. A small end-of-player fragment is still kept as `RemainingBytes` until those bytes are identified.

Skill records include the saved float `level`, a floored `displayLevel` matching the in-game level number, and `accumulator`, which appears to drive progress toward the next displayed level. The accumulator is raw progress input, not a normalized percentage.

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
