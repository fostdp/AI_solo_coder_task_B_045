package supply_analyzer

import (
	"math"
	"testing"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
)

func newTestSupplyAnalyzer(t *testing.T) *SupplyAnalyzer {
	t.Helper()
	cfg := config.DefaultConfig
	return New(&cfg)
}

func standardBattlefield() models.Battlefield {
	return models.Battlefield{
		ID:           1,
		BattleName:   "长平之战",
		Era:          "春秋战国",
		Dynasty:      "战国",
		BattleYear:   -260,
		BelligerentA: "秦军",
		BelligerentB: "赵军",
		TroopA:       50000,
		TroopB:       40000,
		TotalTroops:  90000,
		Lng:          112.5,
		Lat:          35.8,
		Elevation:    1200,
	}
}

func standardRoads() []models.AncientRoad {
	return []models.AncientRoad{
		{
			ID:         1,
			RoadName:   "太行道",
			RoadType:   "官道",
			Importance: 3,
			Coords: [][2]float64{
				{112.0, 35.5},
				{112.3, 35.7},
				{112.6, 35.9},
			},
		},
		{
			ID:         2,
			RoadName:   "滏口陉",
			RoadType:   "陉道",
			Importance: 2,
			Coords: [][2]float64{
				{113.0, 36.0},
				{112.7, 35.85},
			},
		},
		{
			ID:         3,
			RoadName:   "白陉",
			RoadType:   "陉道",
			Importance: 2,
			Coords: [][2]float64{
				{111.8, 35.3},
				{112.1, 35.6},
				{112.4, 35.75},
			},
		},
	}
}

func TestHaversineKm_KnownDistances(t *testing.T) {
	a := newTestSupplyAnalyzer(t)

	cases := []struct {
		name            string
		lng1, lat1      float64
		lng2, lat2      float64
		expected        float64
		tolerance       float64
	}{
		{"Beijing_to_Shanghai", 116.4, 39.9, 121.5, 31.2, 1070.0, 60.0},
		{"Beijing_to_Guangzhou", 116.4, 39.9, 113.3, 23.1, 1880.0, 100.0},
		{"Chengdu_to_Chongqing", 104.1, 30.7, 106.6, 29.6, 270.0, 40.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := a.haversineKm(c.lng1, c.lat1, c.lng2, c.lat2)
			if math.Abs(d-c.expected) > c.tolerance {
				t.Errorf("haversineKm(%v,%v→%v,%v)=%.2f km, expected %.0f±%.0f",
					c.lng1, c.lat1, c.lng2, c.lat2, d, c.expected, c.tolerance)
			}
		})
	}
}

func TestHaversineKm_ZeroDistance(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	d := a.haversineKm(112.5, 35.8, 112.5, 35.8)
	if d > 0.01 {
		t.Errorf("same point should have 0 distance, got %.4f", d)
	}
}

func TestHaversineKm_Symmetry(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	d1 := a.haversineKm(116.4, 39.9, 121.5, 31.2)
	d2 := a.haversineKm(121.5, 31.2, 116.4, 39.9)
	if math.Abs(d1-d2) > 0.01 {
		t.Errorf("haversine should be symmetric: %.4f vs %.4f", d1, d2)
	}
}

func TestBuildNodesAndEdges_ThreeRoads(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()

	nodes, edges := a.buildNodesAndEdges(bf, roads)

	if len(nodes) < 4 {
		t.Errorf("expected at least 4 nodes (battlefield + 3 road coords), got %d", len(nodes))
	}
	if len(edges) < 2 {
		t.Errorf("expected at least 2 edges from 3 roads, got %d", len(edges))
	}
	for i, n := range nodes {
		if n.lng < 70 || n.lng > 140 || n.lat < 10 || n.lat > 60 {
			t.Errorf("node[%d] coordinates out of China range: lng=%.4f lat=%.4f", i, n.lng, n.lat)
		}
	}
	for i, e := range edges {
		if e.fromIdx < 0 || e.fromIdx >= len(nodes) || e.toIdx < 0 || e.toIdx >= len(nodes) {
			t.Errorf("edge[%d] has invalid indices: fromIdx=%d toIdx=%d (nodes=%d)", i, e.fromIdx, e.toIdx, len(nodes))
		}
		if e.length <= 0 {
			t.Errorf("edge[%d] length non-positive: %.4f", i, e.length)
		}
		if e.imp <= 0 {
			t.Errorf("edge[%d] imp non-positive: %d", i, e.imp)
		}
	}
}

func TestBuildNodesAndEdges_EmptyRoads(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	nodes, edges := a.buildNodesAndEdges(bf, nil)

	if len(nodes) != 1 {
		t.Errorf("expected exactly 1 node (battlefield only) with empty roads, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges with empty roads, got %d", len(edges))
	}
}

func TestBuildNodesAndEdges_SinglePointRoad(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := []models.AncientRoad{
		{ID: 99, RoadName: "SinglePoint", Coords: [][2]float64{{112.6, 35.9}}},
	}
	nodes, edges := a.buildNodesAndEdges(bf, roads)
	if len(nodes) < 1 {
		t.Errorf("expected at least 1 node, got %d", len(nodes))
	}
	_ = edges
}

func TestDijkstra_LinearGraph(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	nodes := []struct {
		lng, lat float64
		name     string
	}{
		{116.0, 39.0, "A"},
		{116.5, 39.3, "B"},
		{117.0, 39.6, "C"},
	}
	edges := []roadEdge{
		{id: 1, fromIdx: 0, toIdx: 1, length: 5.0, imp: 1},
		{id: 1, fromIdx: 1, toIdx: 2, length: 3.0, imp: 1},
		{id: 1, fromIdx: 0, toIdx: 2, length: 20.0, imp: 1},
	}
	dist, prev := a.dijkstra(nodes, edges, 0)

	if math.Abs(dist[2]-8.0) > 0.01 {
		t.Errorf("shortest A→C = 8, got %.4f", dist[2])
	}
	if prev[2] != 1 {
		t.Errorf("prev[2] should be 1 (via B), got %d", prev[2])
	}
	if prev[0] != -1 {
		t.Errorf("prev[src] should be -1, got %d", prev[0])
	}
}

func TestDijkstra_DisconnectedGraph(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	nodes := []struct {
		lng, lat float64
		name     string
	}{
		{116.0, 39.0, "A"},
		{117.0, 39.0, "B"},
		{118.0, 39.0, "C"},
	}
	edges := []roadEdge{
		{id: 1, fromIdx: 0, toIdx: 1, length: 1.0, imp: 1},
	}
	dist, _ := a.dijkstra(nodes, edges, 0)
	if dist[2] < 1e17 {
		t.Errorf("disconnected node should have infinite dist, got %.4f", dist[2])
	}
}

func TestDijkstra_SingleNode(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	nodes := []struct {
		lng, lat float64
		name     string
	}{
		{116.0, 39.0, "A"},
	}
	dist, prev := a.dijkstra(nodes, nil, 0)
	if dist[0] != 0 {
		t.Errorf("self distance should be 0, got %.4f", dist[0])
	}
	if prev[0] != -1 {
		t.Errorf("prev[0] should be -1, got %d", prev[0])
	}
}

func TestAnalyzeSupply_NodeGeneration(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()

	result := a.AnalyzeSupply(bf, roads)

	if len(result.NodesA) != 4 {
		t.Errorf("expected 4 nodes for side A, got %d", len(result.NodesA))
	}
	if len(result.NodesB) != 4 {
		t.Errorf("expected 4 nodes for side B, got %d", len(result.NodesB))
	}

	validTypes := map[string]bool{
		"粮仓": true, "武库": true, "兵站": true, "渡口": true,
		"驿站": true, "关卡": true, "集散地": true,
	}
	for i, n := range result.NodesA {
		if !validTypes[n.NodeType] {
			t.Errorf("NodesA[%d] invalid type %q", i, n.NodeType)
		}
		if n.Belligerent != bf.BelligerentA {
			t.Errorf("NodesA[%d] Belligerent=%q, want %q", i, n.Belligerent, bf.BelligerentA)
		}
		if n.Capacity <= 0 {
			t.Errorf("NodesA[%d] capacity %d, want > 0", i, n.Capacity)
		}
	}
	for i, n := range result.NodesB {
		if !validTypes[n.NodeType] {
			t.Errorf("NodesB[%d] invalid type %q", i, n.NodeType)
		}
		if n.Belligerent != bf.BelligerentB {
			t.Errorf("NodesB[%d] Belligerent=%q, want %q", i, n.Belligerent, bf.BelligerentB)
		}
	}
}

func TestAnalyzeSupply_RouteLengthConsistency(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()
	result := a.AnalyzeSupply(bf, roads)

	for ri, r := range append(append([]models.SupplyRoute{}, result.RoutesA...), result.RoutesB...) {
		if len(r.Coords) < 2 {
			t.Errorf("route[%d] %s: need at least 2 coords, got %d", ri, r.RouteName, len(r.Coords))
			continue
		}
		sumDist := 0.0
		for i := 1; i < len(r.Coords); i++ {
			sumDist += a.haversineKm(r.Coords[i-1][0], r.Coords[i-1][1], r.Coords[i][0], r.Coords[i][1])
		}
		ratio := sumDist / r.TotalLengthKm
		if ratio < 0.8 || ratio > 1.2 {
			t.Errorf("route[%d] %s: sum(segment)=%.2f km vs TotalLengthKm=%.2f (ratio %.3f, expected ~1.0)",
				ri, r.RouteName, sumDist, r.TotalLengthKm, ratio)
		}
	}
}

func TestAnalyzeSupply_BottleneckIdentification(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()
	result := a.AnalyzeSupply(bf, roads)

	bnAIds := make(map[int]bool)
	for _, n := range result.BottlenecksA {
		bnAIds[n.ID] = true
		if !n.IsBottleneck {
			t.Errorf("bottleneck node %d not marked IsBottleneck", n.ID)
		}
	}
	for _, n := range result.NodesA {
		if n.IsBottleneck && !bnAIds[n.ID] {
			t.Errorf("node %d is IsBottleneck but missing from BottlenecksA", n.ID)
		}
	}
	for _, r := range result.RoutesA {
		for _, bid := range r.BottleneckIDs {
			if !bnAIds[bid] {
				t.Errorf("route %s references bottleneck %d not in BottlenecksA", r.RouteName, bid)
			}
		}
	}

	bnBIds := make(map[int]bool)
	for _, n := range result.BottlenecksB {
		bnBIds[n.ID] = true
		if !n.IsBottleneck {
			t.Errorf("B bottleneck node %d not marked IsBottleneck", n.ID)
		}
	}
	for _, n := range result.NodesB {
		if n.IsBottleneck && !bnBIds[n.ID] {
			t.Errorf("B node %d is IsBottleneck but missing from BottlenecksB", n.ID)
		}
	}
}

func TestAnalyzeSupply_TimeReachability(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()
	result := a.AnalyzeSupply(bf, roads)

	allRoutes := append(append([]models.SupplyRoute{}, result.RoutesA...), result.RoutesB...)
	for i, r := range allRoutes {
		speedKmh := 15.0 + 5.0
		expectedDays := r.TotalLengthKm / speedKmh / 12.0
		ratio := r.EstTimeDays / expectedDays
		if ratio < 0.7 || ratio > 1.3 {
			t.Errorf("route[%d] %s: EstTimeDays=%.2f, expected≈%.2f (TotalLengthKm=%.2f)",
				i, r.RouteName, r.EstTimeDays, expectedDays, r.TotalLengthKm)
		}
	}
}

func TestAnalyzeSupply_AdvantageScore(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()
	result := a.AnalyzeSupply(bf, roads)

	if result.AdvantageSide != bf.BelligerentA && result.AdvantageSide != bf.BelligerentB {
		t.Errorf("AdvantageSide %q not matching either belligerent", result.AdvantageSide)
	}
	if result.AdvantageScore < 0 {
		t.Errorf("AdvantageScore %.4f should be >= 0", result.AdvantageScore)
	}
}

func TestAnalyzeSupply_EmptyRoads(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	result := a.AnalyzeSupply(bf, nil)

	if len(result.NodesA) != 4 || len(result.NodesB) != 4 {
		t.Errorf("empty roads should still generate 4+4 nodes, got %d+%d", len(result.NodesA), len(result.NodesB))
	}
	if len(result.RoutesA) != 4 || len(result.RoutesB) != 4 {
		t.Errorf("empty roads should still generate 4+4 routes, got %d+%d", len(result.RoutesA), len(result.RoutesB))
	}
}

func TestAnalyzeSupply_ExtremeCoordinates(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	cases := []struct {
		name string
		lng  float64
		lat  float64
	}{
		{"西南边界", 73.5, 18.2},
		{"东北边界", 134.8, 53.5},
		{"中原", 113.6, 34.8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bf := standardBattlefield()
			bf.ID = bf.ID + 100
			bf.Lng = c.lng
			bf.Lat = c.lat
			result := a.AnalyzeSupply(bf, nil)
			if len(result.NodesA) != 4 {
				t.Errorf("%s: expected 4 NodesA, got %d", c.name, len(result.NodesA))
			}
			for _, n := range result.NodesA {
				if n.Lng < 70 || n.Lng > 140 || n.Lat < 10 || n.Lat > 60 {
					t.Errorf("%s: node out of China range: (%.4f,%.4f)", c.name, n.Lng, n.Lat)
				}
			}
		})
	}
}

func TestAnalyzeSupply_BelligerentsSame(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	bf.BelligerentA = "汉军"
	bf.BelligerentB = "汉军"
	result := a.AnalyzeSupply(bf, nil)
	if result.AdvantageSide != "汉军" {
		t.Errorf("both belligerent same, AdvantageSide should be 汉军, got %q", result.AdvantageSide)
	}
}

func TestAnalyzeSupply_ZeroBattlefieldID(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	bf.ID = 0
	result := a.AnalyzeSupply(bf, nil)
	if result.BattlefieldID != 0 {
		t.Errorf("BattlefieldID mismatch: got %d, want 0", result.BattlefieldID)
	}
}

func TestGetLast_Caching(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	result := a.AnalyzeSupply(bf, standardRoads())

	got := a.GetLast(bf.ID)
	if got == nil {
		t.Fatalf("GetLast(%d) returned nil after AnalyzeSupply", bf.ID)
	}
	if got.AdvantageSide != result.AdvantageSide {
		t.Errorf("cached result differs: %q vs %q", got.AdvantageSide, result.AdvantageSide)
	}

	missing := a.GetLast(bf.ID + 9999)
	if missing != nil {
		t.Errorf("GetLast(unknown) should be nil, got %+v", missing)
	}

	neg := a.GetLast(-1)
	if neg != nil {
		t.Errorf("GetLast(-1) should be nil")
	}
}

func TestAnalyzeSupply_Determinism(t *testing.T) {
	a1 := newTestSupplyAnalyzer(t)
	a2 := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()

	r1 := a1.AnalyzeSupply(bf, roads)
	r2 := a2.AnalyzeSupply(bf, roads)

	if len(r1.RoutesA) != len(r2.RoutesA) {
		t.Fatalf("RoutesA count differs: %d vs %d", len(r1.RoutesA), len(r2.RoutesA))
	}
	for i := range r1.RoutesA {
		if r1.RoutesA[i].RouteName != r2.RoutesA[i].RouteName {
			t.Errorf("RoutesA[%d].RouteName differs: %q vs %q", i, r1.RoutesA[i].RouteName, r2.RoutesA[i].RouteName)
		}
		if math.Abs(r1.RoutesA[i].TotalLengthKm-r2.RoutesA[i].TotalLengthKm) > 0.01 {
			t.Errorf("RoutesA[%d].TotalLengthKm differs: %.4f vs %.4f", i, r1.RoutesA[i].TotalLengthKm, r2.RoutesA[i].TotalLengthKm)
		}
	}
	if len(r1.NodesA) != len(r2.NodesA) {
		t.Fatalf("NodesA count differs: %d vs %d", len(r1.NodesA), len(r2.NodesA))
	}
	for i := range r1.NodesA {
		if r1.NodesA[i].NodeType != r2.NodesA[i].NodeType {
			t.Errorf("NodesA[%d].NodeType differs: %q vs %q", i, r1.NodesA[i].NodeType, r2.NodesA[i].NodeType)
		}
	}
}

func TestAnalyzeSupply_RouteEndpointsMatchBattlefield(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()
	result := a.AnalyzeSupply(bf, roads)

	allRoutes := append(append([]models.SupplyRoute{}, result.RoutesA...), result.RoutesB...)
	for i, r := range allRoutes {
		if len(r.Coords) < 2 {
			continue
		}
		last := r.Coords[len(r.Coords)-1]
		dlng := math.Abs(last[0] - bf.Lng)
		dlat := math.Abs(last[1] - bf.Lat)
		if dlng > 0.15 || dlat > 0.15 {
			t.Errorf("route[%d] %s endpoint (%.4f,%.4f) too far from battlefield (%.4f,%.4f)",
				i, r.RouteName, last[0], last[1], bf.Lng, bf.Lat)
		}
	}
}

func TestAnalyzeSupply_RoutesSortedByLength(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()
	result := a.AnalyzeSupply(bf, roads)

	for i := 1; i < len(result.RoutesA); i++ {
		if result.RoutesA[i].TotalLengthKm < result.RoutesA[i-1].TotalLengthKm-0.01 {
			t.Errorf("RoutesA not sorted ascending: [%d]=%.2f > [%d]=%.2f",
				i-1, result.RoutesA[i-1].TotalLengthKm, i, result.RoutesA[i].TotalLengthKm)
		}
	}
	for i := 1; i < len(result.RoutesB); i++ {
		if result.RoutesB[i].TotalLengthKm < result.RoutesB[i-1].TotalLengthKm-0.01 {
			t.Errorf("RoutesB not sorted ascending: [%d]=%.2f > [%d]=%.2f",
				i-1, result.RoutesB[i-1].TotalLengthKm, i, result.RoutesB[i].TotalLengthKm)
		}
	}
}

func TestAnalyzeSupply_EfficiencyInRange(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	roads := standardRoads()
	result := a.AnalyzeSupply(bf, roads)

	all := append(append([]models.SupplyRoute{}, result.RoutesA...), result.RoutesB...)
	for i, r := range all {
		if r.Efficiency <= 0 || r.Efficiency > 1.01 {
			t.Errorf("route[%d] %s Efficiency=%.4f out of (0,1]", i, r.RouteName, r.Efficiency)
		}
	}
}

func TestAnalyzeSupply_BelligerentRoutesCorrect(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	result := a.AnalyzeSupply(bf, nil)

	for i, r := range result.RoutesA {
		if r.Belligerent != bf.BelligerentA {
			t.Errorf("RoutesA[%d] Belligerent=%q, want %q", i, r.Belligerent, bf.BelligerentA)
		}
	}
	for i, r := range result.RoutesB {
		if r.Belligerent != bf.BelligerentB {
			t.Errorf("RoutesB[%d] Belligerent=%q, want %q", i, r.Belligerent, bf.BelligerentB)
		}
	}
}

func TestAnalyzeSupply_NodesACoordinatesEastOfBattlefield(t *testing.T) {
	a := newTestSupplyAnalyzer(t)
	bf := standardBattlefield()
	result := a.AnalyzeSupply(bf, nil)

	for i, n := range result.NodesA {
		if n.Lng < bf.Lng-0.5 {
			t.Errorf("NodesA[%d] Lng=%.4f should be near/east of battlefield Lng=%.4f", i, n.Lng, bf.Lng)
		}
	}
	for i, n := range result.NodesB {
		if n.Lng > bf.Lng+0.5 {
			t.Errorf("NodesB[%d] Lng=%.4f should be near/west of battlefield Lng=%.4f", i, n.Lng, bf.Lng)
		}
	}
}
