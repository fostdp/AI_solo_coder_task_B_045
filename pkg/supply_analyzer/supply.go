package supply_analyzer

import (
	"math"
	"sort"
	"sync"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
)

type SupplyAnalyzer struct {
	cfg *config.ModelConfig
	mu  sync.RWMutex

	lastResult map[int]*models.SupplyAnalysis
}

func New(cfg *config.ModelConfig) *SupplyAnalyzer {
	return &SupplyAnalyzer{
		cfg:        cfg,
		lastResult: make(map[int]*models.SupplyAnalysis),
	}
}

func (a *SupplyAnalyzer) haversineKm(lng1, lat1, lng2, lat2 float64) float64 {
	const R = 6371.0
	toRad := math.Pi / 180.0
	dLat := (lat2 - lat1) * toRad
	dLng := (lng2 - lng1) * toRad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*toRad)*math.Cos(lat2*toRad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

type roadEdge struct {
	id      int
	fromIdx int
	toIdx   int
	length  float64
	imp     int
}

func (a *SupplyAnalyzer) buildNodesAndEdges(
	bf models.Battlefield,
	roads []models.AncientRoad,
) ([]struct {
	lng, lat float64
	name     string
}, []roadEdge) {
	nodes := make([]struct {
		lng, lat float64
		name     string
	}, 0)
	edges := make([]roadEdge, 0)

	nodes = append(nodes, struct {
		lng, lat float64
		name     string
	}{bf.Lng, bf.Lat, "战场" + bf.BattleName})

	for _, road := range roads {
		if len(road.Coords) < 2 {
			continue
		}
		startIdx := len(nodes)
		nodes = append(nodes, struct {
			lng, lat float64
			name     string
		}{road.Coords[0][0], road.Coords[0][1], road.RoadName + "_起"})

		prevIdx := startIdx
		for i := 1; i < len(road.Coords); i++ {
			curIdx := len(nodes)
			nodes = append(nodes, struct {
				lng, lat float64
				name     string
			}{road.Coords[i][0], road.Coords[i][1], road.RoadName + "_p"})

			lng1, lat1 := road.Coords[i-1][0], road.Coords[i-1][1]
			lng2, lat2 := road.Coords[i][0], road.Coords[i][1]
			dist := a.haversineKm(lng1, lat1, lng2, lat2)
			edges = append(edges, roadEdge{
				id:      road.ID,
				fromIdx: prevIdx,
				toIdx:   curIdx,
				length:  math.Max(0.5, dist),
				imp:     road.Importance,
			})
			edges = append(edges, roadEdge{
				id:      road.ID,
				fromIdx: curIdx,
				toIdx:   prevIdx,
				length:  math.Max(0.5, dist),
				imp:     road.Importance,
			})
			prevIdx = curIdx
		}
	}

	for i := 1; i < len(nodes); i++ {
		dist := a.haversineKm(nodes[0].lng, nodes[0].lat, nodes[i].lng, nodes[i].lat)
		if dist < 20 {
			edges = append(edges, roadEdge{
				id:      -1,
				fromIdx: 0,
				toIdx:   i,
				length:  dist * 1.2,
				imp:     3,
			})
			edges = append(edges, roadEdge{
				id:      -1,
				fromIdx: i,
				toIdx:   0,
				length:  dist * 1.2,
				imp:     3,
			})
		}
	}

	return nodes, edges
}

func (a *SupplyAnalyzer) dijkstra(
	nodes []struct {
		lng, lat float64
		name     string
	},
	edges []roadEdge,
	src int,
) ([]float64, []int) {
	n := len(nodes)
	dist := make([]float64, n)
	prev := make([]int, n)
	visited := make([]bool, n)
	for i := range dist {
		dist[i] = 1e18
		prev[i] = -1
	}
	dist[src] = 0

	for iter := 0; iter < n; iter++ {
		u := -1
		best := 1e18
		for i := 0; i < n; i++ {
			if !visited[i] && dist[i] < best {
				best = dist[i]
				u = i
			}
		}
		if u < 0 {
			break
		}
		visited[u] = true

		for _, e := range edges {
			if e.fromIdx != u {
				continue
			}
			cost := e.length / float64(e.imp)
			if dist[u]+cost < dist[e.toIdx] {
				dist[e.toIdx] = dist[u] + cost
				prev[e.toIdx] = u
			}
		}
	}

	return dist, prev
}

func (a *SupplyAnalyzer) reconstructPath(prev []int, src, dst int) []int {
	path := make([]int, 0)
	cur := dst
	for cur != -1 && cur != src {
		path = append(path, cur)
		cur = prev[cur]
	}
	if cur == src {
		path = append(path, src)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func (a *SupplyAnalyzer) terrainPenalty(lng, lat, elev float64) (float64, string) {
	basePenalty := 1.0
	desc := "平原通衢"
	switch {
	case elev > 3000:
		basePenalty = 3.5
		desc = "高原险阻"
	case elev > 1500:
		basePenalty = 2.2
		desc = "山地崎岖"
	case elev > 500:
		basePenalty = 1.4
		desc = "丘陵起伏"
	}
	if lat < 25 && lng > 100 {
		basePenalty *= 1.2
		desc = desc + "+岭南湿热"
	}
	if lng < 100 && lat > 38 {
		basePenalty *= 1.15
		desc = desc + "+西北干旱"
	}
	return basePenalty, desc
}

func (a *SupplyAnalyzer) archaeoEvidence(era, nodeType string) string {
	evidence := map[string]map[string]string{
		"春秋战国": {"粮仓": "敖仓、陈留仓遗址群", "渡口": "孟津、白马津古渡", "关卡": "函谷关、武关遗址"},
		"秦汉":     {"粮仓": "太仓、华仓考古遗址", "驿站": "悬泉置驿遗址", "关卡": "阳关、玉门关遗址"},
		"三国两晋南北朝": {"渡口": "赤壁古渡、瓜洲渡", "武库": "洛阳武库遗址"},
		"隋唐五代": {"粮仓": "含嘉仓、洛口仓遗址", "渡口": "汴河渡口群", "集散地": "扬州、益州商埠"},
		"宋辽金元": {"驿站": "鸡鸣驿、急递铺遗址", "关卡": "雁门关、居庸关", "渡口": "采石矶、瓜洲渡"},
		"明清":     {"粮仓": "京通二仓遗址", "关卡": "山海关、嘉峪关", "集散地": "临清、苏州钞关"},
	}
	m, ok := evidence[era]
	if !ok {
		m = evidence["秦汉"]
	}
	if ev, ok2 := m[nodeType]; ok2 {
		return ev
	}
	return "同代考古遗址旁证"
}

func (a *SupplyAnalyzer) generateSupplyNodes(
	bf models.Battlefield,
	roads []models.AncientRoad,
	side string,
	seed int,
	count int,
) ([]models.SupplyNode, [][]int) {
	nodes := make([]models.SupplyNode, 0, count)
	nodeRoadIDs := make([][]int, 0, count)

	centerLng := bf.Lng
	centerLat := bf.Lat
	baseDir := 1.0
	if side == "B" {
		baseDir = -1.0
	}

	nodeTypes := []string{"粮仓", "武库", "兵站", "渡口", "驿站", "关卡", "集散地"}
	useArchaeo := len(roads) < 2

	for i := 0; i < count; i++ {
		distFactor := 0.2 + float64(i)*0.08
		lngOffset := baseDir * distFactor * (0.8 + pseudoRandFloat(seed+i*7)*0.4)
		latOffset := pseudoRandFloat(seed+i*13-3)*0.15
		ndLng := centerLng + lngOffset
		ndLat := centerLat + latOffset

		elev := bf.Elevation + pseudoRandFloat(seed+i*29)*200
		terrainPen, terrainDesc := a.terrainPenalty(ndLng, ndLat, elev)
		_ = terrainPen

		capacity := 5000 + pseudoRandInt(seed*3+i*11)%30000
		isBottle := pseudoRandFloat(seed+i*19) < 0.3
		throughput := float64(capacity) * (0.4 + pseudoRandFloat(seed+i*23)*0.5)

		nearRoadIDs := make([]int, 0)
		for _, road := range roads {
			if len(road.Coords) == 0 {
				continue
			}
			midIdx := len(road.Coords) / 2
			d := a.haversineKm(ndLng, ndLat, road.Coords[midIdx][0], road.Coords[midIdx][1])
			if d < 15 {
				nearRoadIDs = append(nearRoadIDs, road.ID)
				if len(nearRoadIDs) >= 2 {
					break
				}
			}
		}

		nt := nodeTypes[i%len(nodeTypes)]
		archaeoStr := ""
		if useArchaeo {
			archaeoStr = a.archaeoEvidence(bf.Era, nt)
		}

		nodes = append(nodes, models.SupplyNode{
			ID:                     seed*100 + i + 1,
			NodeName:               side + "方-" + nt,
			NodeType:               nt,
			Belligerent:            side,
			Lng:                    math.Round(ndLng*10000) / 10000,
			Lat:                    math.Round(ndLat*10000) / 10000,
			Capacity:               capacity,
			IsBottleneck:           isBottle,
			Throughput:             math.Round(throughput*100) / 100,
			RoadIDs:                nearRoadIDs,
			ArchaeologicalEvidence: archaeoStr,
			TerrainConstraint:      terrainDesc,
		})
		nodeRoadIDs = append(nodeRoadIDs, nearRoadIDs)
	}

	return nodes, nodeRoadIDs
}

func (a *SupplyAnalyzer) buildRoutes(
	bf models.Battlefield,
	supplyNodes []models.SupplyNode,
	roads []models.AncientRoad,
	side string,
	seed int,
) []models.SupplyRoute {
	routes := make([]models.SupplyRoute, 0)
	routeNamePrefix := side + "方补给线"
	useArchaeo := len(roads) < 2

	for i, node := range supplyNodes {
		coords := make([][2]float64, 0)
		coords = append(coords, [2]float64{node.Lng, node.Lat})

		intermediateCount := 2 + pseudoRandInt(seed+i*5)%2
		curLng := node.Lng
		curLat := node.Lat
		terrainPenSum := 0.0
		for j := 0; j < intermediateCount; j++ {
			stepLng := (bf.Lng - node.Lng) / float64(intermediateCount+1)
			stepLat := (bf.Lat - node.Lat) / float64(intermediateCount+1)
			nextLng := curLng + stepLng + pseudoRandFloat(seed*7+i*3+j)*0.02
			nextLat := curLat + stepLat + pseudoRandFloat(seed*11+i*5+j)*0.015
			sampleElev := bf.Elevation + pseudoRandFloat(seed*13+i*7+j*3)*150
			tp, _ := a.terrainPenalty(nextLng, nextLat, sampleElev)
			terrainPenSum += tp
			coords = append(coords, [2]float64{
				math.Round(nextLng*10000) / 10000,
				math.Round(nextLat*10000) / 10000,
			})
			curLng = nextLng
			curLat = nextLat
		}

		coords = append(coords, [2]float64{bf.Lng, bf.Lat})

		totalLen := 0.0
		for k := 1; k < len(coords); k++ {
			totalLen += a.haversineKm(coords[k-1][0], coords[k-1][1], coords[k][0], coords[k][1])
		}
		avgTerrainPen := 1.0
		if intermediateCount > 0 {
			avgTerrainPen = terrainPenSum / float64(intermediateCount)
		}

		capacity := node.Capacity
		speedKmh := (15.0 + pseudoRandFloat(seed+i)*10) / avgTerrainPen
		timeDays := (totalLen / speedKmh) / 12
		efficiency := (0.5 + pseudoRandFloat(seed+i*17)*0.4) / math.Sqrt(avgTerrainPen)
		if efficiency > 1.0 {
			efficiency = 1.0
		}

		bottleneckIDs := make([]int, 0)
		if node.IsBottleneck {
			bottleneckIDs = append(bottleneckIDs, node.ID)
		}

		archaeoSupport := ""
		if useArchaeo {
			archaeoSupport = "基于" + bf.Era + "同期古道走向推断"
		}

		routes = append(routes, models.SupplyRoute{
			ID:             seed*200 + i + 1,
			RouteName:      routeNamePrefix + "-" + fmtIdx(i),
			Belligerent:    side,
			Coords:         coords,
			RoadSegments:   node.RoadIDs,
			TotalLengthKm:  math.Round(totalLen*100) / 100,
			Capacity:       capacity,
			EstTimeDays:    math.Round(timeDays*100) / 100,
			Efficiency:     math.Round(efficiency*10000) / 10000,
			Nodes:          []int{node.ID},
			BottleneckIDs:  bottleneckIDs,
			TerrainPenalty: math.Round(avgTerrainPen*10000) / 10000,
			ArchaeoSupport: archaeoSupport,
		})
	}

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].TotalLengthKm < routes[j].TotalLengthKm
	})

	return routes
}

func fmtIdx(i int) string {
	return runeToStr(rune('A' + i))
}

func runeToStr(r rune) string {
	return string(r)
}

func (a *SupplyAnalyzer) AnalyzeSupply(
	bf models.Battlefield,
	roads []models.AncientRoad,
) models.SupplyAnalysis {
	seed := bf.ID

	nodesA, _ := a.generateSupplyNodes(bf, roads, bf.BelligerentA, seed, 4)
	nodesB, _ := a.generateSupplyNodes(bf, roads, bf.BelligerentB, -seed, 4)

	routesA := a.buildRoutes(bf, nodesA, roads, bf.BelligerentA, seed)
	routesB := a.buildRoutes(bf, nodesB, roads, bf.BelligerentB, -seed)

	bottlenecksA := make([]models.SupplyNode, 0)
	for _, n := range nodesA {
		if n.IsBottleneck {
			bottlenecksA = append(bottlenecksA, n)
		}
	}
	bottlenecksB := make([]models.SupplyNode, 0)
	for _, n := range nodesB {
		if n.IsBottleneck {
			bottlenecksB = append(bottlenecksB, n)
		}
	}

	scoreA := 0.0
	for _, r := range routesA {
		scoreA += r.Efficiency*float64(r.Capacity)/10000 - r.EstTimeDays*0.1
	}
	scoreB := 0.0
	for _, r := range routesB {
		scoreB += r.Efficiency*float64(r.Capacity)/10000 - r.EstTimeDays*0.1
	}
	scoreA = math.Round(scoreA*100) / 100
	scoreB = math.Round(scoreB*100) / 100

	advantage := bf.BelligerentA
	advScore := scoreA - scoreB
	if scoreB > scoreA {
		advantage = bf.BelligerentB
		advScore = scoreB - scoreA
	}

	terrainApplied := false
	archaeoCount := 0
	for _, n := range append(append([]models.SupplyNode{}, nodesA...), nodesB...) {
		if n.TerrainConstraint != "" {
			terrainApplied = true
		}
	}
	for _, n := range nodesA {
		if n.ArchaeologicalEvidence != "" {
			archaeoCount++
		}
	}
	for _, n := range nodesB {
		if n.ArchaeologicalEvidence != "" {
			archaeoCount++
		}
	}

	result := models.SupplyAnalysis{
		BattlefieldID:            bf.ID,
		BelligerentA:             bf.BelligerentA,
		BelligerentB:             bf.BelligerentB,
		RoutesA:                   routesA,
		RoutesB:                   routesB,
		NodesA:                    nodesA,
		NodesB:                    nodesB,
		BottlenecksA:               bottlenecksA,
		BottlenecksB:               bottlenecksB,
		AdvantageSide:             advantage,
		AdvantageScore:            math.Round(advScore*100) / 100,
		TerrainConstraintApplied: terrainApplied,
		ArchaeoEvidenceCount:    archaeoCount,
	}

	a.mu.Lock()
	a.lastResult[bf.ID] = &result
	a.mu.Unlock()

	return result
}

func (a *SupplyAnalyzer) GetLast(bfID int) *models.SupplyAnalysis {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastResult[bfID]
}

func pseudoRandInt(seed int) int {
	s := uint64(seed*2654435761 + 1)
	s = s*6364136223846793005 + 1442695040888963407
	v := int(s >> 33)
	if v < 0 {
		v = -v
	}
	return v
}

func pseudoRandFloat(seed int) float64 {
	s := uint64(seed*2654435761 + 1009)
	s = s*6364136223846793005 + 1442695040888963407
	return float64(s>>11) / (1 << 53)
}
