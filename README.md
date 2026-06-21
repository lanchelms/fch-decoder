# fch-decoder

Go decoder for current Valheim `.fch` character files.

The decoder currently reads:

- file/version header and the 72-byte footer/hash block
- leading player stat table with `PlayerStatType` names
- the embedded gzip map block metadata
- player name, player ID, cheat-use flag, and character creation timestamp
- known world time, known world-key time history, known command, enemy, material, and recipe stat arrays
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

`knownWorlds` and `knownWorldKeys` are accumulated elapsed seconds. Valheim adds
the seconds since the last save/load to the current world name and to each
observed global-key state. They are history/telemetry entries, not current world
settings.

Current Valheim decompilation writes three player tail floats after player
custom data: stamina, max eitr, and eitr. The bundled fixtures contain the first
two tail floats, so `eitr` remains zero for those files.

Skill records include the saved float `level`, a floored `displayLevel` matching the in-game level number, and `accumulator`, which appears to drive progress toward the next displayed level. The accumulator is raw progress input, not a normalized percentage.

## CLI

```sh
go run ./cmd/fchdump 'testdata/Steam_76561198018104185_bortson.fch'
```

## Prometheus exporter

```sh
go run ./cmd/fchprom -dir "$HOME/.config/unity3d/IronGate/Valheim/characters_local" -addr :9108
```

The exporter serves `/metrics` and emits:

- `valheim_character_skills{player,skill}`
- `valheim_character_crafting{player,recipe}`
- `valheim_character_enemies{player,enemy}`
- `valheim_character_stats{player,stat}`

Only current `.fch` files are loaded; `.fch.old` and `backup_auto-*.fch` files
are ignored. Each scrape rediscovers the directory and decodes files through a
bounded worker pool. Decoded metrics are cached briefly by default so close
scrapes do not repeatedly parse the same files:

```sh
go run ./cmd/fchprom -dir testdata -addr :9108 -workers 4 -cache-ttl 5s
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
