package valheim

import (
	"sort"
	"strings"
)

var skillNames = map[int32]string{
	0:   "None",
	1:   "Swords",
	2:   "Knives",
	3:   "Clubs",
	4:   "Polearms",
	5:   "Spears",
	6:   "Blocking",
	7:   "Axes",
	8:   "Bows",
	9:   "ElementalMagic",
	10:  "BloodMagic",
	11:  "Unarmed",
	12:  "Pickaxes",
	13:  "WoodCutting",
	14:  "Crossbows",
	100: "Jump",
	101: "Sneak",
	102: "Run",
	103: "Swim",
	104: "Fishing",
	105: "Cooking",
	106: "Farming",
	107: "Crafting",
	108: "Dodge",
	110: "Ride",
	999: "All",
}

func skillName(skillType int32) string {
	return skillNames[skillType]
}

func SkillTypeByName(name string) (int32, bool) {
	for skillType, skillName := range skillNames {
		if strings.EqualFold(skillName, name) {
			return skillType, true
		}
	}
	return 0, false
}

// SkillNames returns known skill names sorted alphabetically.
func SkillNames() []string {
	names := make([]string, 0, len(skillNames))
	for _, name := range skillNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Constructors keep the 105-entry version 43 stat table.
var playerStatNames = currentPlayerStatNames[:105]

var currentPlayerStatNames = []string{
	"Deaths",
	"CraftsOrUpgrades",
	"Builds",
	"Jumps",
	"Cheats",
	"EnemyHits",
	"EnemyKills",
	"EnemyKillsLastHits",
	"PlayerHits",
	"PlayerKills",
	"HitsTakenEnemies",
	"HitsTakenPlayers",
	"ItemsPickedUp",
	"Crafts",
	"Upgrades",
	"PortalsUsed",
	"DistanceTraveled",
	"DistanceWalk",
	"DistanceRun",
	"DistanceSail",
	"DistanceAir",
	"TimeInBase",
	"TimeOutOfBase",
	"Sleep",
	"ItemStandUses",
	"ArmorStandUses",
	"WorldLoads",
	"TreeChops",
	"Tree",
	"TreeTier0",
	"TreeTier1",
	"TreeTier2",
	"TreeTier3",
	"TreeTier4",
	"TreeTier5",
	"LogChops",
	"Logs",
	"MineHits",
	"Mines",
	"MineTier0",
	"MineTier1",
	"MineTier2",
	"MineTier3",
	"MineTier4",
	"MineTier5",
	"RavenHits",
	"RavenTalk",
	"RavenAppear",
	"CreatureTamed",
	"FoodEaten",
	"SkeletonSummons",
	"ArrowsShot",
	"TombstonesOpenedOwn",
	"TombstonesOpenedOther",
	"TombstonesFit",
	"DeathByUndefined",
	"DeathByEnemyHit",
	"DeathByPlayerHit",
	"DeathByFall",
	"DeathByDrowning",
	"DeathByBurning",
	"DeathByFreezing",
	"DeathByPoisoned",
	"DeathBySmoke",
	"DeathByWater",
	"DeathByEdgeOfWorld",
	"DeathByImpact",
	"DeathByCart",
	"DeathByTree",
	"DeathBySelf",
	"DeathByStructural",
	"DeathByTurret",
	"DeathByBoat",
	"DeathByStalagtite",
	"DoorsOpened",
	"DoorsClosed",
	"BeesHarvested",
	"SapHarvested",
	"TurretAmmoAdded",
	"TurretTrophySet",
	"TrapArmed",
	"TrapTriggered",
	"PlaceStacks",
	"PortalDungeonIn",
	"PortalDungeonOut",
	"BossKills",
	"BossLastHits",
	"SetGuardianPower",
	"SetPowerEikthyr",
	"SetPowerElder",
	"SetPowerBonemass",
	"SetPowerModer",
	"SetPowerYagluth",
	"SetPowerQueen",
	"SetPowerAshlands",
	"SetPowerDeepNorth",
	"UseGuardianPower",
	"UsePowerEikthyr",
	"UsePowerElder",
	"UsePowerBonemass",
	"UsePowerModer",
	"UsePowerYagluth",
	"UsePowerQueen",
	"UsePowerAshlands",
	"UsePowerDeepNorth",
	"DeathByCatapult",
	"DeathByCinderFire",
	"DeathByAshlandsOcean",
	"DeathByIncinerator",
	"CraftFood",
	"CraftFoodBonus",
	"CraftGrill",
	"CraftGrillBurnt",
	"CraftGrillBonus",
	"CraftWeapon",
	"CraftArmor",
	"CraftTrinket",
	"CraftAmmo",
	"CraftMaterial",
	"CraftTool",
	"CraftTorch",
	"CraftBait",
	"CraftOther",
	"HarvestCrop",
	"HarvestBerry",
	"HarvestMushroom",
	"HarvestVine",
	"HarvestBonus",
	"ConsecutiveDaysSurvived",
	"ConsecutiveDaysSurvivedMax",
	"MaxBuildingHeight",
	"MaxBuildingHeightWorld",
	"MaxComfort",
	"TreasureBuriedFound",
	"TreasureDungeonFound",
	"TreasureLocationFound",
	"LeviathanSink",
	"LavaLeviathanSink",
	"ExploreNorth",
	"ExploreSouth",
	"ExploreEast",
	"ExploreWest",
	"ExploreNorthNoMap",
	"ExploreSouthNoMap",
	"ExploreEastNoMap",
	"ExploreWestNoMap",
	"BossKillMultiplayer",
	"BossKillSolo",
	"VillagePointsMax",
	"DeepestDungeon",
	"DistanceSailHelm",
	"PlayerSpawn",
	"DeathByTreeTier0",
	"DeathByTreeTier1",
	"DeathByTreeTier2",
	"DeathByTreeTier3",
	"DeathByTreeTier4",
	"DeathByTreeTier5",
	"FishHooked",
	"FishLost",
	"FishCaught",
	"FishCaughtTier0",
	"FishCaughtTier1",
	"FishCaughtTier2",
	"FishCaughtTier3",
	"FishCaughtTier4",
	"FishCaughtTier5",
	"FishCaughtTier6",
	"BuiltPieces",
	"BuiltPiecesNoDebt",
	"BuildPiecesRemoved",
	"BuildClusterMisc",
	"BuildClusterCrafting",
	"BuildClusterBuilding",
	"BuildClusterFloor",
	"BuildClusterWall",
	"BuildClusterRoof",
	"BuildClusterArchitecture",
	"BuildClusterFurniture",
	"BuildClusterLighting",
	"BuildClusterDecor",
	"BuildClusterStorage",
	"BuildClusterTransport",
	"BuildClusterFood",
	"BuildClusterMeads",
	"BuildClusterFeasts",
	"BuildClusterDefense",
	"BuildClusterStacks",
	"BuildClusterStairs",
	"BuildClusterDoors",
	"BuildClusterSeasonal",
	"TamedPetting",
	"TamedCommand",
	"TreeFir",
	"TreeOak",
	"TreePine",
	"TreeAshlands",
	"TreeYggdrasilShoot",
	"TreeSwamp",
	"TreeBeech",
	"TreeBirch",
	"TreeSnowFir",
	"TreeSnowPine",
	"DeathByDrawBridge",
	"DeathByAshlandsLava",
}

func playerStatName(index int) string {
	if index < 0 || index >= len(currentPlayerStatNames) {
		return ""
	}
	return currentPlayerStatNames[index]
}

func PlayerStatIndexByName(name string) (int, bool) {
	for i, statName := range currentPlayerStatNames {
		if strings.EqualFold(statName, name) {
			return i, true
		}
	}
	return 0, false
}

// PlayerStatNames returns known player stat names in saved stat order.
func PlayerStatNames() []string {
	return append([]string(nil), currentPlayerStatNames...)
}
