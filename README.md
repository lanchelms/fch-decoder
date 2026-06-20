# fch-decoder

Go decoder for current Valheim `.fch` character files.

The decoder currently reads:

- file/version header and the 72-byte footer/hash block
- leading player stat table with `PlayerStatType` names
- the embedded gzip map block metadata
- player name, player ID, cheat-use flag, and character creation timestamp
- known world, known world key, known command, enemy, material, and recipe stat arrays
- known material and recipe lists
- known stations, shown tutorials, unique unlocks, trophies, known biomes, player known texts, hair/beard style, colors, and model index
- player health/stamina/eitr state and equipped guardian power
- inventory item records, including item custom data, world level, and picked-up flag
- active food records
- player skill records
- player custom data

`materialStats` and `recipeStats` are the raw per-item stat counters saved in
the character file. They are not current inventory contents, and evidence from
stack-producing items suggests they should not be interpreted as lifetime item
quantities either. For example, crafted arrows and food can have stat values far
below the number of items produced because the game appears to count stat events
rather than stack amounts.

Current Valheim decompilation writes three player tail floats after player
custom data: stamina, max eitr, and eitr. The bundled fixtures contain the first
two tail floats, so `eitr` remains zero for those files.

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
