package battlefield_loader

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	"ancient-battlefield/pkg/models"
)

type Loader struct {
	mu       sync.RWMutex
	dataPath string

	Battlefields []models.Battlefield
	Roads        []models.AncientRoad
	Rivers       []models.AncientRiver
	DEM          []models.DEMTile
}

func New(dataPath string) *Loader {
	if dataPath == "" {
		dataPath = filepath.Join("web", "data", "data.json")
	}
	return &Loader{dataPath: dataPath}
}

func (l *Loader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.dataPath)
	if err == nil {
		var pkg struct {
			Battlefields []models.Battlefield   `json:"battlefields"`
			Roads        []models.AncientRoad   `json:"roads"`
			Rivers       []models.AncientRiver  `json:"rivers"`
			DEM          []models.DEMTile       `json:"dem_tiles"`
		}
		if jerr := json.Unmarshal(data, &pkg); jerr == nil {
			l.Battlefields = pkg.Battlefields
			l.Roads = pkg.Roads
			l.Rivers = pkg.Rivers
			l.DEM = pkg.DEM
			return nil
		}
	}
	return l.generateFallback()
}

func (l *Loader) generateFallback() error {
	rng := rand.New(rand.NewSource(2024))

	eras := []struct {
		name string
		min  int
		max  int
	}{
		{"春秋战国", -770, -221},
		{"秦汉", -221, 220},
		{"三国两晋南北朝", 220, 581},
		{"隋唐五代", 581, 960},
		{"宋辽金元", 960, 1368},
		{"明清", 1368, 1911},
	}
	sides := [][]string{
		{"秦国", "赵国"}, {"汉军", "楚军"}, {"魏军", "蜀军"},
		{"唐军", "突厥"}, {"宋军", "辽军"}, {"明军", "元军"},
		{"清军", "明军"}, {"匈奴", "汉朝"}, {"吴国", "越国"},
	}
	terrains := []string{"山地", "平原", "河谷", "关隘"}
	outcomes := []string{"进攻方胜", "防守方胜", "僵持"}

	l.Battlefields = make([]models.Battlefield, 800)
	for i := 0; i < 800; i++ {
		eraIdx := weightedEraIdx(rng)
		era := eras[eraIdx]
		year := era.min + rng.Intn(era.max-era.min)
		lng, lat := randomChinaCoord(rng)

		elevation := mockElevation(lng, lat, rng)
		sidePair := sides[rng.Intn(len(sides))]
		troops1 := 5000 + rng.Intn(200000)
		troops2 := 5000 + rng.Intn(200000)

		distRoad := 2.0 + rng.Float64()*80.0
		distRiver := 3.0 + rng.Float64()*120.0

		l.Battlefields[i] = models.Battlefield{
			ID:            int64(i + 1),
			BattleName:    fmt.Sprintf("%s_%d号战役", era.name, i+1),
			Era:           era.name,
			Year:          int32(year),
			BelligerentA:  sidePair[0],
			BelligerentB:  sidePair[1],
			TroopsA:       int32(troops1),
			TroopsB:       int32(troops2),
			TotalTroops:   int32(troops1 + troops2),
			Lng:           lng,
			Lat:           lat,
			Elevation:     int32(elevation),
			TerrainType:   terrains[rng.Intn(len(terrains))],
			Outcome:       outcomes[rng.Intn(len(outcomes))],
			DistToRoad:    distRoad,
			DistToRiver:   distRiver,
		}
	}

	l.Roads = generateRoads(rng)
	l.Rivers = generateRivers(rng)
	l.DEM = generateDEM(rng)
	return nil
}

func weightedEraIdx(rng *rand.Rand) int {
	cdf := []float64{0.15, 0.20, 0.18, 0.15, 0.18, 0.14}
	r := rng.Float64()
	var acc float64
	for i, w := range cdf {
		acc += w
		if r <= acc {
			return i
		}
	}
	return len(cdf) - 1
}

func randomChinaCoord(rng *rand.Rand) (float64, float64) {
	for {
		lng := 73.0 + rng.Float64()*62.0
		lat := 18.0 + rng.Float64()*36.0
		if !isOcean(lng, lat) {
			return lng, lat
		}
	}
}

func isOcean(lng, lat float64) bool {
	if lat < 23.0 && lng > 119.0 {
		if lng > 120.0+lat*0.5 {
			return true
		}
	}
	return false
}

func mockElevation(lng, lat float64, rng *rand.Rand) int {
	var base float64
	switch {
	case lng < 95.0:
		base = 3500
	case lat > 30.0 && lng < 105.0:
		base = 2000
	case lat > 40.0 && lng < 110.0:
		base = 1200
	case lat < 25.0:
		base = 200
	default:
		base = 600
	}
	base += (rng.Float64() - 0.5) * 400
	return int(math.Max(0, base))
}

func generateRoads(rng *rand.Rand) []models.AncientRoad {
	roads := []struct {
		name     string
		waypoint [][2]float64
	}{
		{"丝绸之路东段", [][2]float64{{108.0, 34.0}, {101.0, 36.0}, {95.0, 39.0}, {87.0, 43.0}, {79.0, 41.0}}},
		{"大运河", [][2]float64{{116.0, 39.0}, {117.0, 36.0}, {119.0, 32.0}, {120.0, 30.0}, {120.0, 25.0}}},
		{"蜀道", [][2]float64{{108.0, 34.0}, {106.0, 33.0}, {104.0, 32.0}, {104.0, 30.0}, {104.0, 28.0}}},
		{"茶马古道", [][2]float64{{103.0, 25.0}, {101.0, 27.0}, {99.0, 29.0}, {97.0, 30.0}, {91.0, 30.0}}},
		{"秦驰道", [][2]float64{{108.0, 34.0}, {114.0, 36.0}, {118.0, 37.0}, {121.0, 40.0}}},
		{"岭南新道", [][2]float64{{113.0, 25.0}, {112.0, 27.0}, {114.0, 29.0}, {116.0, 31.0}}},
	}
	out := make([]models.AncientRoad, 60)
	for i := 0; i < 60; i++ {
		if i < len(roads) {
			pts := make([][2]float64, len(roads[i].waypoint))
			copy(pts, roads[i].waypoint)
			out[i] = models.AncientRoad{
				ID:       int64(i + 1),
				RoadName: roads[i].name,
				Coords:   pts,
			}
		} else {
			lng := 75 + rng.Float64()*55
			lat := 20 + rng.Float64()*30
			n := 3 + rng.Intn(5)
			pts := make([][2]float64, n)
			pts[0] = [2]float64{lng, lat}
			for j := 1; j < n; j++ {
				pts[j] = [2]float64{
					pts[j-1][0] + (rng.Float64()-0.5)*6,
					pts[j-1][1] + (rng.Float64()-0.5)*4,
				}
			}
			out[i] = models.AncientRoad{
				ID:       int64(i + 1),
				RoadName: fmt.Sprintf("古道_%d", i+1),
				Coords:   pts,
			}
		}
	}
	return out
}

func generateRivers(rng *rand.Rand) []models.AncientRiver {
	rivers := []struct {
		name string
		pts  [][2]float64
	}{
		{"黄河", [][2]float64{{96.0, 35.0}, {102.0, 36.0}, {106.0, 38.0}, {108.0, 40.0},
			{111.0, 39.0}, {113.0, 35.0}, {117.0, 37.0}, {119.0, 38.0}}},
		{"长江", [][2]float64{{90.0, 30.0}, {95.0, 31.0}, {100.0, 28.0}, {104.0, 29.0},
			{108.0, 31.0}, {112.0, 30.0}, {116.0, 30.0}, {121.0, 31.0}}},
		{"淮河", [][2]float64{{113.0, 33.0}, {116.0, 33.0}, {119.0, 33.5}, {122.0, 33.0}}},
		{"珠江", [][2]float64{{104.0, 25.0}, {108.0, 24.0}, {112.0, 23.5}, {113.0, 23.0}}},
	}
	out := make([]models.AncientRiver, 25)
	for i := 0; i < 25; i++ {
		if i < len(rivers) {
			pts := make([][2]float64, len(rivers[i].pts))
			copy(pts, rivers[i].pts)
			out[i] = models.AncientRiver{
				ID:         int64(i + 1),
				RiverName:  rivers[i].name,
				Coords:     pts,
				Importance: "major",
			}
		} else {
			lng := 75 + rng.Float64()*55
			lat := 20 + rng.Float64()*30
			n := 3 + rng.Intn(5)
			pts := make([][2]float64, n)
			pts[0] = [2]float64{lng, lat}
			for j := 1; j < n; j++ {
				pts[j] = [2]float64{
					pts[j-1][0] + (rng.Float64())*4,
					pts[j-1][1] + (rng.Float64()-0.5)*3,
				}
			}
			out[i] = models.AncientRiver{
				ID:         int64(i + 1),
				RiverName:  fmt.Sprintf("支流_%d", i+1),
				Coords:     pts,
				Importance: "minor",
			}
		}
	}
	return out
}

func generateDEM(rng *rand.Rand) []models.DEMTile {
	out := make([]models.DEMTile, 0, 6*8)
	for lng := 75.0; lng <= 135.0; lng += 10.0 {
		for lat := 18.0; lat <= 54.0; lat += 10.0 {
			elev := int32(mockElevation(lng, lat, rng))
			grid := make([][]int32, 5)
			for y := 0; y < 5; y++ {
				grid[y] = make([]int32, 5)
				for x := 0; x < 5; x++ {
					grid[y][x] = elev + int32(rng.Intn(400)-200)
				}
			}
			out = append(out, models.DEMTile{
				ID:         int64(len(out) + 1),
				TileZ:      5,
				TileX:      int32((lng - 73.0) / 10),
				TileY:      int32((54.0 - lat) / 10),
				MinElev:    int32(math.Max(0, float64(elev-300))),
				MaxElev:    elev + 300,
				GridSize:   5,
				HeightGrid: grid,
			})
		}
	}
	return out
}

func (l *Loader) GetBattlefields() []models.Battlefield {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Battlefields
}

func (l *Loader) GetRoads() []models.AncientRoad {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Roads
}

func (l *Loader) GetRivers() []models.AncientRiver {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Rivers
}

func (l *Loader) GetDEM() []models.DEMTile {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.DEM
}
