package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	MinLng = 73.0
	MaxLng = 135.0
	MinLat = 18.0
	MaxLat = 54.0
)

type Battlefield struct {
	ID              int        `json:"id"`
	BattleName      string     `json:"battle_name"`
	Dynasty         string     `json:"dynasty"`
	Era             string     `json:"era"`
	BattleYear      int        `json:"battle_year"`
	BelligerentA    string     `json:"belligerent_a"`
	BelligerentB    string     `json:"belligerent_b"`
	TroopA          int        `json:"troop_a"`
	TroopB          int        `json:"troop_b"`
	TotalTroops     int        `json:"total_troops"`
	TerrainType     string     `json:"terrain_type"`
	Result          string     `json:"result"`
	Lng             float64    `json:"lng"`
	Lat             float64    `json:"lat"`
	Elevation       float64    `json:"elevation"`
	DistanceToRiver float64    `json:"distance_to_river"`
	DistanceToRoad  float64    `json:"distance_to_road"`
}

type AncientRoad struct {
	ID         int          `json:"id"`
	RoadName   string       `json:"road_name"`
	RoadType   string       `json:"road_type"`
	Dynasty    string       `json:"dynasty"`
	Importance int          `json:"importance"`
	Coords     [][2]float64 `json:"coords"`
}

type AncientRiver struct {
	ID        int          `json:"id"`
	RiverName string       `json:"river_name"`
	RiverType string       `json:"river_type"`
	Coords    [][2]float64 `json:"coords"`
}

type DEMTile struct {
	TileX    int        `json:"tile_x"`
	TileY    int        `json:"tile_y"`
	Zoom     int        `json:"zoom"`
	MinElev  float64    `json:"min_elev"`
	MaxElev  float64    `json:"max_elev"`
	GridSize int        `json:"grid_size"`
	HeightGrid [][]int  `json:"height_grid"`
}

type FullDataset struct {
	Battlefields []Battlefield  `json:"battlefields"`
	Roads        []AncientRoad  `json:"roads"`
	Rivers       []AncientRiver `json:"rivers"`
	DEMTiles     []DEMTile      `json:"dem_tiles"`
	DEMGrid      [][]struct {
		Lng  float64 `json:"lng"`
		Lat  float64 `json:"lat"`
		Elev float64 `json:"elev"`
	} `json:"dem_grid"`
}

var eraDefinitions = []struct {
	Name       string
	Dynasties  []string
	YearRange  [2]int
	Weight     float64
}{
	{"春秋战国", []string{"春秋", "战国"}, [2]int{-770, -221}, 0.17},
	{"秦汉", []string{"秦", "西汉", "东汉"}, [2]int{-221, 220}, 0.18},
	{"三国两晋南北朝", []string{"三国", "西晋", "东晋", "南北朝"}, [2]int{220, 589}, 0.16},
	{"隋唐五代", []string{"隋", "唐", "五代十国"}, [2]int{581, 960}, 0.15},
	{"宋辽金元", []string{"北宋", "南宋", "辽", "金", "元"}, [2]int{960, 1368}, 0.18},
	{"明清", []string{"明", "清"}, [2]int{1368, 1912}, 0.16},
}

var (
	prefixes = []string{"牧野", "长平", "巨鹿", "赤壁", "淝水", "官渡", "夷陵", "街亭",
		"雁门", "函谷", "虎牢", "襄阳", "彭城", "垓下", "定军", "祁山", "潼关", "井陉",
		"马陵", "桂陵", "城濮", "鄢郢", "即墨", "肥下", "番吾", "邯郸", "涿鹿", "鸣条"}
	suffixes = []string{"之战", "大捷", "保卫战", "攻坚战", "突围战", "伏击战", "遭遇战", "决战"}
	factions = []string{"秦", "楚", "齐", "燕", "赵", "魏", "韩", "晋", "吴", "越",
		"汉", "蜀", "隋", "唐", "宋", "辽", "金", "元", "明", "清",
		"匈奴", "突厥", "吐蕃", "契丹", "女真", "蒙古", "义军"}
	terrainTypes = []string{"山地", "平原", "河谷", "关隘"}
	results      = []string{"A方胜", "B方胜", "双方议和", "僵持不下"}
	roadTypes    = []string{"驿道", "栈道", "漕运", "官道", "古道"}

	historicRoads = []struct {
		Name      string
		Type      string
		Dynasty   string
		Waypoints [][2]float64
	}{
		{"丝绸之路东段", "驿道", "汉", [][2]float64{{108, 34}, {101, 36}, {95, 39}, {87, 43}}},
		{"大运河", "漕运", "隋", [][2]float64{{116, 39}, {117, 36}, {119, 32}, {120, 30}}},
		{"蜀道", "栈道", "秦", [][2]float64{{108, 34}, {106, 33}, {104, 30}, {104, 28}}},
		{"茶马古道", "古道", "唐", [][2]float64{{103, 25}, {101, 27}, {99, 29}, {91, 30}}},
		{"秦驰道", "官道", "秦", [][2]float64{{108, 34}, {114, 36}, {118, 37}}},
		{"岭南新道", "驿道", "汉", [][2]float64{{113, 25}, {112, 27}, {114, 29}, {116, 31}}},
	}

	historicRivers = []struct {
		Name string
		Type string
		Pts  [][2]float64
	}{
		{"黄河", "河流", [][2]float64{{96, 35}, {102, 36}, {106, 38}, {108, 40},
			{111, 39}, {113, 35}, {117, 37}, {119, 38}}},
		{"长江", "河流", [][2]float64{{90, 30}, {95, 31}, {100, 28}, {104, 29},
			{108, 31}, {112, 30}, {116, 30}, {121, 31}}},
		{"淮河", "河流", [][2]float64{{113, 33}, {116, 33}, {119, 33.5}, {122, 33}}},
		{"珠江", "河流", [][2]float64{{104, 25}, {108, 24}, {112, 23.5}, {113, 23}}},
	}
)

func main() {
	count := flag.Int("count", 800, "战场数量")
	eraFilter := flag.String("era", "all", "年代筛选: all,春秋战国,秦汉,三国两晋南北朝,隋唐五代,宋辽金元,明清")
	terrainFilter := flag.String("terrain", "all", "地形筛选: all,山地,平原,河谷,关隘")
	demRes := flag.Float64("dem-resolution", 1.0, "DEM分辨率(度)")
	output := flag.String("output", "./web/data/data.json", "输出文件路径")
	seed := flag.Int64("seed", 0, "随机种子(0=当前时间)")
	roads := flag.Int("roads", 60, "道路数量")
	rivers := flag.Int("rivers", 25, "河流数量")
	verbose := flag.Bool("v", false, "详细输出")
	flag.Parse()

	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(*seed))

	validEras := parseFilter(*eraFilter, eraNames())
	validTerrains := parseFilter(*terrainFilter, terrainTypes)

	bfs := generateBattlefields(rng, *count, validEras, validTerrains)
	rdList := generateRoadList(rng, *roads)
	rvList := generateRiverList(rng, *rivers)
	demTiles := generateDEMTiles(rng, *demRes)

	dataset := map[string]interface{}{
		"battlefields": bfs,
		"roads":        rdList,
		"rivers":       rvList,
		"dem_tiles":    demTiles,
	}

	if err := os.MkdirAll(dirOf(*output), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建目录失败: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON序列化失败: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 数据生成完成\n")
	fmt.Printf("  战场遗址: %d 个", len(bfs))
	if *eraFilter != "all" {
		fmt.Printf(" (年代: %s)", *eraFilter)
	}
	if *terrainFilter != "all" {
		fmt.Printf(" (地形: %s)", *terrainFilter)
	}
	fmt.Printf("\n  古代道路: %d 条\n", len(rdList))
	fmt.Printf("  河流水系: %d 条\n", len(rvList))
	fmt.Printf("  DEM栅格: %d 个瓦片 (分辨率: %.1f°)\n", len(demTiles), *demRes)
	fmt.Printf("  随机种子: %d\n", *seed)
	fmt.Printf("  输出: %s\n", *output)

	if *verbose {
		eraCount := make(map[string]int)
		terrCount := make(map[string]int)
		for _, b := range bfs {
			eraCount[b.Era]++
			terrCount[b.TerrainType]++
		}
		fmt.Println("\n  年代分布:")
		for _, e := range eraNames() {
			fmt.Printf("    %s: %d\n", e, eraCount[e])
		}
		fmt.Println("  地形分布:")
		for _, t := range terrainTypes {
			fmt.Printf("    %s: %d\n", t, terrCount[t])
		}
	}
}

func eraNames() []string {
	names := make([]string, len(eraDefinitions))
	for i, e := range eraDefinitions {
		names[i] = e.Name
	}
	return names
}

func parseFilter(flag string, all []string) map[string]bool {
	if flag == "all" {
		m := make(map[string]bool, len(all))
		for _, s := range all {
			m[s] = true
		}
		return m
	}
	m := make(map[string]bool)
	for _, s := range strings.Split(flag, ",") {
		trimmed := strings.TrimSpace(s)
		for _, a := range all {
			if a == trimmed {
				m[trimmed] = true
			}
		}
	}
	return m
}

func generateBattlefields(rng *rand.Rand, n int, eras, terrains map[string]bool) []Battlefield {
	result := make([]Battlefield, 0, n)
	id := 1
	for len(result) < n {
		eraIdx := weightedEraIdx(rng)
		eraDef := eraDefinitions[eraIdx]
		if !eras[eraDef.Name] {
			continue
		}
		lng := MinLng + rng.Float64()*(MaxLng-MinLng)
		lat := MinLat + rng.Float64()*(MaxLat-MinLat)
		if lat < 23 && lng > 119 && lng > 120+lat*0.5 {
			continue
		}

		elev := mockElevation(lng, lat, rng)
		terrain := pickTerrain(rng, elev, terrains)
		if terrain == "" {
			continue
		}

		dynasty := eraDef.Dynasties[rng.Intn(len(eraDef.Dynasties))]
		year := eraDef.YearRange[0] + rng.Intn(eraDef.YearRange[1]-eraDef.YearRange[0])
		troopA := 5000 + rng.Intn(200000)
		troopB := 5000 + rng.Intn(200000)

		result = append(result, Battlefield{
			ID:              id,
			BattleName:      prefixes[rng.Intn(len(prefixes))] + suffixes[rng.Intn(len(suffixes))],
			Dynasty:         dynasty,
			Era:             eraDef.Name,
			BattleYear:      year,
			BelligerentA:    factions[rng.Intn(len(factions))],
			BelligerentB:    factions[rng.Intn(len(factions))],
			TroopA:          troopA,
			TroopB:          troopB,
			TotalTroops:     troopA + troopB,
			TerrainType:     terrain,
			Result:          results[rng.Intn(len(results))],
			Lng:             math.Round(lng*1000) / 1000,
			Lat:             math.Round(lat*1000) / 1000,
			Elevation:       math.Round(elev*10) / 10,
			DistanceToRiver: math.Round((3+rng.Float64()*120)*10) / 10,
			DistanceToRoad:  math.Round((2+rng.Float64()*80)*10) / 10,
		})
		id++
	}
	return result
}

func weightedEraIdx(rng *rand.Rand) int {
	r := rng.Float64()
	var acc float64
	for i, e := range eraDefinitions {
		acc += e.Weight
		if r <= acc {
			return i
		}
	}
	return len(eraDefinitions) - 1
}

func mockElevation(lng, lat float64, rng *rand.Rand) float64 {
	var base float64
	switch {
	case lng < 95:
		base = 3500
	case lat > 30 && lng < 105:
		base = 2000
	case lat > 40 && lng < 110:
		base = 1200
	case lat < 25:
		base = 200
	default:
		base = 600
	}
	base += (rng.Float64() - 0.5) * 400
	return math.Max(0, base)
}

func pickTerrain(rng *rand.Rand, elev float64, valid map[string]bool) string {
	var preferred string
	if elev > 1500 {
		preferred = "山地"
	} else if elev < 100 {
		preferred = "平原"
	}
	if valid[preferred] && rng.Float64() < 0.6 {
		return preferred
	}
	candidates := make([]string, 0, len(terrainTypes))
	for _, t := range terrainTypes {
		if valid[t] {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	return candidates[rng.Intn(len(candidates))]
}

func generateRoadList(rng *rand.Rand, n int) []AncientRoad {
	roads := make([]AncientRoad, 0, n)
	id := 1
	for _, hr := range historicRoads {
		pts := make([][2]float64, len(hr.Waypoints))
		copy(pts, hr.Waypoints)
		roads = append(roads, AncientRoad{
			ID: id, RoadName: hr.Name, RoadType: hr.Type,
			Dynasty: hr.Dynasty, Importance: 5, Coords: pts,
		})
		id++
	}
	for len(roads) < n {
		startLng := MinLng + rng.Float64()*(MaxLng-MinLng)
		startLat := MinLat + rng.Float64()*(MaxLat-MinLat)
		numPts := 4 + rng.Intn(8)
		coords := make([][2]float64, numPts)
		coords[0] = [2]float64{startLng, startLat}
		for j := 1; j < numPts; j++ {
			coords[j] = [2]float64{
				coords[j-1][0] + (rng.Float64()-0.5)*6,
				coords[j-1][1] + (rng.Float64()-0.5)*4,
			}
		}
		era := eraDefinitions[rng.Intn(len(eraDefinitions))]
		roads = append(roads, AncientRoad{
			ID: id, RoadName: fmt.Sprintf("%s古道%d号线", era.Dynasties[0], id),
			RoadType: roadTypes[rng.Intn(len(roadTypes))],
			Dynasty:  era.Dynasties[rng.Intn(len(era.Dynasties))],
			Importance: 1 + rng.Intn(5), Coords: coords,
		})
		id++
	}
	return roads
}

func generateRiverList(rng *rand.Rand, n int) []AncientRiver {
	rivers := make([]AncientRiver, 0, n)
	id := 1
	for _, hr := range historicRivers {
		pts := make([][2]float64, len(hr.Pts))
		copy(pts, hr.Pts)
		rivers = append(rivers, AncientRiver{
			ID: id, RiverName: hr.Name, RiverType: hr.Type, Coords: pts,
		})
		id++
	}
	for len(rivers) < n {
		startLng := MinLng + rng.Float64()*(MaxLng-MinLng)
		startLat := MinLat + rng.Float64()*(MaxLat-MinLat)
		numPts := 5 + rng.Intn(10)
		coords := make([][2]float64, numPts)
		coords[0] = [2]float64{startLng, startLat}
		for j := 1; j < numPts; j++ {
			coords[j] = [2]float64{
				coords[j-1][0] + rng.Float64()*3,
				coords[j-1][1] + (rng.Float64()-0.5)*2,
			}
		}
		rType := "河流"
		if id > n-5 {
			rType = "湖泊"
		}
		rivers = append(rivers, AncientRiver{
			ID: id, RiverName: fmt.Sprintf("支流_%d", id),
			RiverType: rType, Coords: coords,
		})
		id++
	}
	return rivers
}

func generateDEMTiles(rng *rand.Rand, res float64) []DEMTile {
	var tiles []DEMTile
	step := 10.0
	for lng := MinLng; lng < MaxLng; lng += step {
		for lat := MinLat; lat < MaxLat; lat += step {
			centerElev := mockElevation(lng+step/2, lat+step/2, rng)
			gridSize := int(step / res)
			if gridSize < 2 {
				gridSize = 2
			}
			if gridSize > 20 {
				gridSize = 20
			}
			grid := make([][]int, gridSize)
			for y := 0; y < gridSize; y++ {
				grid[y] = make([]int, gridSize)
				for x := 0; x < gridSize; x++ {
					localLng := lng + float64(x)*step/float64(gridSize)
					localLat := lat + float64(y)*step/float64(gridSize)
					grid[y][x] = int(mockElevation(localLng, localLat, rng))
				}
			}
			tileX := int((lng - MinLng) / step)
			tileY := int((MaxLat - lat) / step)
			tiles = append(tiles, DEMTile{
				TileX: tileX, TileY: tileY, Zoom: int(step),
				MinElev: math.Max(0, centerElev-300),
				MaxElev: centerElev + 300,
				GridSize: gridSize, HeightGrid: grid,
			})
		}
	}
	return tiles
}

func dirOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		idx = strings.LastIndex(path, "\\")
	}
	if idx < 0 {
		return "."
	}
	return path[:idx]
}
