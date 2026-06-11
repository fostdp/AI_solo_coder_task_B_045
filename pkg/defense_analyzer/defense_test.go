package defense_analyzer

import (
	"math"
	"strings"
	"testing"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
)

func newTestAnalyzer(t *testing.T) *DefenseAnalyzer {
	t.Helper()
	cfg := config.DefaultConfig
	return New(&cfg)
}

func standardBattlefield() models.Battlefield {
	return models.Battlefield{
		ID:              1,
		BattleName:      "长平之战",
		Dynasty:         "战国",
		Era:             "春秋战国",
		BattleYear:      -260,
		BelligerentA:    "秦军",
		BelligerentB:    "赵军",
		TroopA:          50000,
		TroopB:          40000,
		TotalTroops:     90000,
		TerrainType:     "山地",
		Lng:             112.5,
		Lat:             35.8,
		Elevation:       1200,
		DistanceToRiver: 5.0,
		DistanceToRoad:  3.0,
	}
}

func validStructureTypes() map[string]bool {
	return map[string]bool{
		"城墙": true, "关隘": true, "堡垒": true, "烽火台": true,
		"要塞": true, "寨堡": true, "护城河": true,
	}
}

func TestGenerateMilitaryStructures_CountAndBasicFields(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	structures := a.GenerateMilitaryStructures(bf)

	if len(structures) < 3 || len(structures) > 5 {
		t.Errorf("expected 3-5 structures, got %d", len(structures))
	}

	validTypes := validStructureTypes()
	for i, s := range structures {
		if s.StructureType == "" {
			t.Errorf("structure[%d]: StructureType is empty", i)
		}
		if !validTypes[s.StructureType] {
			t.Errorf("structure[%d]: invalid StructureType %q", i, s.StructureType)
		}
		if s.StructureName == "" {
			t.Errorf("structure[%d]: StructureName is empty", i)
		}
		if s.HeightM <= 0 {
			t.Errorf("structure[%d]: HeightM=%.2f, want > 0", i, s.HeightM)
		}
		if s.ThicknessM <= 0 {
			t.Errorf("structure[%d]: ThicknessM=%.2f, want > 0", i, s.ThicknessM)
		}
		if s.ID == 0 {
			t.Errorf("structure[%d]: ID is zero", i)
		}
		if s.BattlefieldID != bf.ID {
			t.Errorf("structure[%d]: BattlefieldID=%d, want %d", i, s.BattlefieldID, bf.ID)
		}
	}
}

func TestGenerateMilitaryStructures_CoordinatesWithinRange(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	structures := a.GenerateMilitaryStructures(bf)

	for i, s := range structures {
		dlng := math.Abs(s.Lng - bf.Lng)
		dlat := math.Abs(s.Lat - bf.Lat)
		if dlng > 0.2 {
			t.Errorf("structure[%d]: Lng offset %.4f exceeds 0.2", i, dlng)
		}
		if dlat > 0.2 {
			t.Errorf("structure[%d]: Lat offset %.4f exceeds 0.2", i, dlat)
		}
	}
}

func TestGenerateMilitaryStructures_CoordsSet(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	structures := a.GenerateMilitaryStructures(bf)

	for i, s := range structures {
		if len(s.Coords) == 0 {
			t.Errorf("structure[%d] %q: Coords is empty, expected polygon rings", i, s.StructureType)
			continue
		}
		for ri, ring := range s.Coords {
			if len(ring) < 3 {
				t.Errorf("structure[%d] ring[%d]: has %d points, need >= 3 for polygon", i, ri, len(ring))
			}
		}
	}
}

func TestGenerateMilitaryStructures_Determinism(t *testing.T) {
	a1 := newTestAnalyzer(t)
	a2 := newTestAnalyzer(t)
	bf := standardBattlefield()

	s1 := a1.GenerateMilitaryStructures(bf)
	s2 := a2.GenerateMilitaryStructures(bf)

	if len(s1) != len(s2) {
		t.Fatalf("structure count differs: %d vs %d", len(s1), len(s2))
	}
	for i := range s1 {
		if s1[i].StructureType != s2[i].StructureType {
			t.Errorf("structure[%d] type differs: %q vs %q", i, s1[i].StructureType, s2[i].StructureType)
		}
		if math.Abs(s1[i].HeightM-s2[i].HeightM) > 0.01 {
			t.Errorf("structure[%d] height differs: %.4f vs %.4f", i, s1[i].HeightM, s2[i].HeightM)
		}
	}
}

func TestGenerateMilitaryStructures_SortedByHeight(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	structures := a.GenerateMilitaryStructures(bf)

	for i := 1; i < len(structures); i++ {
		if structures[i].HeightM > structures[i-1].HeightM+0.01 {
			t.Errorf("structures not sorted by HeightM desc: [%d]=%.2f < [%d]=%.2f",
				i-1, structures[i-1].HeightM, i, structures[i].HeightM)
		}
	}
}

func TestStructuralScore_ZeroHeight(t *testing.T) {
	a := newTestAnalyzer(t)
	s := models.MilitaryStructure{
		HeightM:    0,
		LengthM:    0,
		ThicknessM: 0,
		Material:   "夯土",
		TowerCount: 0,
	}
	score := a.structuralScore(s)
	if score < 0 || score > 1 {
		t.Errorf("zero-dim structural score %.4f out of [0,1]", score)
	}
	if score > 0.1 {
		t.Errorf("zero-dim structural score %.4f should be very low", score)
	}
}

func TestStructuralScore_TallThickStructure(t *testing.T) {
	a := newTestAnalyzer(t)
	s := models.MilitaryStructure{
		HeightM:    20,
		LengthM:    10000,
		ThicknessM: 12,
		Material:   "条石包砖",
		TowerCount: 30,
	}
	score := a.structuralScore(s)
	if score < 0.85 {
		t.Errorf("tall/thick/good-material structure score %.4f, want >= 0.85", score)
	}
	if score > 1.0001 {
		t.Errorf("structural score %.4f exceeds 1.0", score)
	}
}

func TestStructuralScore_MaterialOrder(t *testing.T) {
	a := newTestAnalyzer(t)
	base := models.MilitaryStructure{HeightM: 10, LengthM: 1000, ThicknessM: 5, TowerCount: 10}
	materials := []string{"夯土木栅", "夯土", "砖石", "夯土包砖", "条石", "条石包砖"}
	scores := make([]float64, len(materials))
	for i, m := range materials {
		s := base
		s.Material = m
		scores[i] = a.structuralScore(s)
	}
	for i := 1; i < len(scores); i++ {
		if scores[i] < scores[i-1]-0.001 {
			t.Errorf("material quality order broken: %q(%.4f) >= %q(%.4f) expected",
				materials[i-1], scores[i-1], materials[i], scores[i])
		}
	}
}

func TestTopographicScore_ElevationDifference(t *testing.T) {
	a := newTestAnalyzer(t)
	bfHigh := standardBattlefield()
	bfHigh.Lng = 90
	bfHigh.Lat = 35

	bfLow := standardBattlefield()
	bfLow.Lng = 120
	bfLow.Lat = 22

	s := models.MilitaryStructure{Lng: 112.5, Lat: 35.8}

	scoreHigh := a.topographicScore(s, bfHigh)
	scoreLow := a.topographicScore(s, bfLow)

	t.Logf("high-ground score=%.4f, low-ground score=%.4f", scoreHigh, scoreLow)
	if scoreHigh < 0 || scoreHigh > 1 || scoreLow < 0 || scoreLow > 1 {
		t.Errorf("topographic scores out of [0,1]: high=%.4f low=%.4f", scoreHigh, scoreLow)
	}
}

func TestTopographicScore_RiverAndRoadBonus(t *testing.T) {
	a := newTestAnalyzer(t)
	s := models.MilitaryStructure{Lng: 112.5, Lat: 35.8}

	bfNear := standardBattlefield()
	bfNear.DistanceToRiver = 1.0
	bfNear.DistanceToRoad = 1.0

	bfFar := standardBattlefield()
	bfFar.DistanceToRiver = 10.0
	bfFar.DistanceToRoad = 10.0

	scoreNear := a.topographicScore(s, bfNear)
	scoreFar := a.topographicScore(s, bfFar)

	if scoreNear < scoreFar-0.001 {
		t.Errorf("near river/road should give >= score: near=%.4f far=%.4f", scoreNear, scoreFar)
	}
}

func TestMakeRecommendations_Conditions(t *testing.T) {
	a := newTestAnalyzer(t)

	t.Run("low visibility triggers tower rec", func(t *testing.T) {
		recs := a.makeRecommendations(nil, 0.8, 0.8, 0.5)
		found := false
		for _, r := range recs {
			if strings.Contains(r, "瞭望塔") {
				found = true
			}
		}
		if !found {
			t.Errorf("low visibility (0.5) should recommend 瞭望塔, got %v", recs)
		}
	})

	t.Run("high-risk blindzone triggers outpost rec", func(t *testing.T) {
		bzs := []models.DefenseBlindZone{{Direction: "正北", RiskLevel: "极高"}}
		recs := a.makeRecommendations(bzs, 0.9, 0.9, 0.95)
		found := false
		for _, r := range recs {
			if strings.Contains(r, "正北") && strings.Contains(r, "极高") {
				found = true
			}
		}
		if !found {
			t.Errorf("高-risk blindzone should mention direction and risk level, got %v", recs)
		}
	})

	t.Run("low structural triggers reinforcement", func(t *testing.T) {
		recs := a.makeRecommendations(nil, 0.3, 0.8, 0.9)
		found := false
		for _, r := range recs {
			if strings.Contains(r, "加固") || strings.Contains(r, "结构") {
				found = true
			}
		}
		if !found {
			t.Errorf("low structural should recommend 加固, got %v", recs)
		}
	})

	t.Run("low topographic triggers terrain rec", func(t *testing.T) {
		recs := a.makeRecommendations(nil, 0.9, 0.2, 0.9)
		found := false
		for _, r := range recs {
			if strings.Contains(r, "地形") {
				found = true
			}
		}
		if !found {
			t.Errorf("low topographic should recommend 地形, got %v", recs)
		}
	})

	t.Run("all good gives maintenance rec", func(t *testing.T) {
		recs := a.makeRecommendations(nil, 0.9, 0.9, 0.95)
		if len(recs) == 0 {
			t.Fatalf("empty recs for good defense")
		}
		found := false
		for _, r := range recs {
			if strings.Contains(r, "良好") || strings.Contains(r, "维护") {
				found = true
			}
		}
		if !found {
			t.Errorf("good defense should mention 良好/维护, got %v", recs)
		}
	})
}

func TestEvaluateStructure_ScoreRanges(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	structures := a.GenerateMilitaryStructures(bf)

	for i, s := range structures {
		eval := a.EvaluateStructure(s, bf, 8.0)

		if eval.StructureID != s.ID {
			t.Errorf("eval[%d]: StructureID=%d, want %d", i, eval.StructureID, s.ID)
		}
		if eval.StructureName != s.StructureName {
			t.Errorf("eval[%d]: StructureName=%q, want %q", i, eval.StructureName, s.StructureName)
		}

		scores := map[string]float64{
			"Overall":      eval.OverallScore,
			"Visibility":   eval.VisibilityScore,
			"Structural":   eval.StructuralScore,
			"Topographic":  eval.TopographicScore,
		}
		for name, sc := range scores {
			if sc < -0.001 || sc > 1.001 {
				t.Errorf("eval[%d]: %sScore=%.4f out of [0,1]", i, name, sc)
			}
		}

		expectedOverall := math.Round((eval.VisibilityScore*0.4+eval.StructuralScore*0.3+eval.TopographicScore*0.3)*10000) / 10000
		if math.Abs(eval.OverallScore-expectedOverall) > 0.001 {
			t.Errorf("eval[%d]: OverallScore=%.4f, expected weighted sum %.4f (vis=%.4f str=%.4f topo=%.4f)",
				i, eval.OverallScore, expectedOverall,
				eval.VisibilityScore, eval.StructuralScore, eval.TopographicScore)
		}

		if math.Abs(eval.AvgVisibilityPct-eval.VisibilityScore) > 0.001 {
			t.Errorf("eval[%d]: AvgVisibilityPct=%.4f != VisibilityScore=%.4f",
				i, eval.AvgVisibilityPct, eval.VisibilityScore)
		}

		if len(eval.Recommendations) == 0 {
			t.Errorf("eval[%d]: Recommendations should not be empty", i)
		}
		if len(eval.Recommendations) > 5 {
			t.Errorf("eval[%d]: too many recommendations %d (>5)", i, len(eval.Recommendations))
		}
	}
}

func TestEvaluateStructure_TallerStructuresBetterVisibility(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	heights := []float64{2, 5, 10, 15, 20, 30}
	scores := make([]float64, len(heights))

	for i, h := range heights {
		s := models.MilitaryStructure{
			ID:             i + 1,
			BattlefieldID:   bf.ID,
			StructureType:   "关隘",
			StructureName:   "test",
			HeightM:         h,
			LengthM:         500,
			ThicknessM:      5,
			Material:        "条石",
			GateCount:       2,
			TowerCount:      8,
			Lng:             bf.Lng,
			Lat:             bf.Lat,
		}
		eval := a.EvaluateStructure(s, bf, 8.0)
		scores[i] = eval.VisibilityScore
	}

	t.Logf("height→visibility: %v", scores)
	nonDecreasing := true
	for i := 1; i < len(scores); i++ {
		if scores[i] < scores[i-1]-0.02 {
			nonDecreasing = false
			t.Logf("  decrease at i=%d: %.4f -> %.4f", i, scores[i-1], scores[i])
		}
	}
	if !nonDecreasing {
		t.Errorf("taller structures should generally have >= visibility scores: %v", scores)
	}
}

func TestEvaluateStructure_BlindZoneRiskLevels(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	structures := a.GenerateMilitaryStructures(bf)

	validRisk := map[string]bool{"低": true, "中": true, "高": true, "极高": true}

	for _, s := range structures {
		eval := a.EvaluateStructure(s, bf, 8.0)

		totalArea := 0.0
		for bi, bz := range eval.BlindZones {
			if bz.VisibilityPct >= 0.5 {
				t.Errorf("blindZone[%d] %s: VisibilityPct=%.4f >= 0.5, should be < 0.5",
					bi, bz.Direction, bz.VisibilityPct)
			}
			if !validRisk[bz.RiskLevel] {
				t.Errorf("blindZone[%d]: invalid RiskLevel %q", bi, bz.RiskLevel)
			}
			if bz.AreaKm2 <= 0 {
				t.Errorf("blindZone[%d]: AreaKm2=%.4f, want > 0", bi, bz.AreaKm2)
			}
			if bz.MaxDistanceKm <= 0 {
				t.Errorf("blindZone[%d]: MaxDistanceKm=%.4f, want > 0", bi, bz.MaxDistanceKm)
			}
			if len(bz.Coords) == 0 || len(bz.Coords[0]) < 3 {
				t.Errorf("blindZone[%d]: Coords empty or too few points", bi)
			}
			totalArea += bz.AreaKm2
		}

		if math.Abs(totalArea-eval.TotalBlindAreaKm2) > 0.01 {
			t.Errorf("TotalBlindAreaKm2=%.4f, sum of zones=%.4f", eval.TotalBlindAreaKm2, totalArea)
		}
	}
}

func TestEvaluateStructure_BlindZonesSortedByArea(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	structures := a.GenerateMilitaryStructures(bf)

	for si, s := range structures {
		eval := a.EvaluateStructure(s, bf, 8.0)
		for i := 1; i < len(eval.BlindZones); i++ {
			if eval.BlindZones[i].AreaKm2 > eval.BlindZones[i-1].AreaKm2+0.001 {
				t.Errorf("structure[%d] blindZones not sorted desc by area: [%d]=%.4f < [%d]=%.4f",
					si, i-1, eval.BlindZones[i-1].AreaKm2, i, eval.BlindZones[i].AreaKm2)
			}
		}
	}
}

func TestEvaluateStructure_ViewshedSampleZeroDefaults(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	s := a.GenerateMilitaryStructures(bf)[0]

	eval0 := a.EvaluateStructure(s, bf, 0)
	eval8 := a.EvaluateStructure(s, bf, 8.0)

	if eval0.ViewshedSampleKm != 8.0 {
		t.Errorf("viewshedSampleKm=0 should default to 8.0, got %.2f", eval0.ViewshedSampleKm)
	}

	evalNeg := a.EvaluateStructure(s, bf, -5)
	if evalNeg.ViewshedSampleKm != 8.0 {
		t.Errorf("negative viewshedSampleKm should default to 8.0, got %.2f", evalNeg.ViewshedSampleKm)
	}
	_ = eval8
}

func TestEvaluateStructure_ExtremeCoordinates(t *testing.T) {
	a := newTestAnalyzer(t)
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
			bf.Lng = c.lng
			bf.Lat = c.lat
			structures := a.GenerateMilitaryStructures(bf)
			for _, s := range structures {
				eval := a.EvaluateStructure(s, bf, 8.0)
				if eval.OverallScore < 0 || eval.OverallScore > 1 {
					t.Errorf("%s: OverallScore=%.4f out of range", c.name, eval.OverallScore)
				}
			}
		})
	}
}

func TestEvaluateStructure_ZeroHeightStructure(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	s := models.MilitaryStructure{
		ID:             99,
		BattlefieldID:   bf.ID,
		StructureType:   "城墙",
		StructureName:   "零高城墙",
		HeightM:         0,
		LengthM:         0,
		ThicknessM:      0,
		Material:        "夯土",
		GateCount:       0,
		TowerCount:      0,
		Lng:             bf.Lng,
		Lat:             bf.Lat,
	}
	eval := a.EvaluateStructure(s, bf, 8.0)
	if eval.StructureID != 99 {
		t.Errorf("zero-height structure eval failed, got StructureID=%d", eval.StructureID)
	}
	if eval.OverallScore < 0 || eval.OverallScore > 1 {
		t.Errorf("zero-height OverallScore=%.4f out of [0,1]", eval.OverallScore)
	}
}

func TestGetLast_Caching(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	s := a.GenerateMilitaryStructures(bf)[0]
	eval := a.EvaluateStructure(s, bf, 8.0)

	got := a.GetLast(s.ID)
	if got == nil {
		t.Fatalf("GetLast(%d) nil after EvaluateStructure", s.ID)
	}
	if got.OverallScore != eval.OverallScore {
		t.Errorf("cached OverallScore %.4f != eval %.4f", got.OverallScore, eval.OverallScore)
	}

	missing := a.GetLast(999999)
	if missing != nil {
		t.Errorf("GetLast(unknown) should be nil")
	}
}

func TestHaversineKm_ZeroDistance(t *testing.T) {
	a := newTestAnalyzer(t)
	d := a.haversineKm(112.5, 35.8, 112.5, 35.8)
	if d > 0.01 {
		t.Errorf("same point distance should be ~0, got %.4f", d)
	}
}

func TestHaversineKm_ReasonableDistance(t *testing.T) {
	a := newTestAnalyzer(t)
	d := a.haversineKm(112.5, 35.8, 112.6, 35.8)
	if d < 5 || d > 20 {
		t.Errorf("0.1 deg lng at ~36 lat should be ~9km, got %.2f", d)
	}
}

func TestGenerateMilitaryStructures_NoNegativeDimensions(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	structures := a.GenerateMilitaryStructures(bf)
	for i, s := range structures {
		if s.HeightM < 0 {
			t.Errorf("structure[%d] HeightM=%.2f negative", i, s.HeightM)
		}
		if s.LengthM < 0 {
			t.Errorf("structure[%d] LengthM=%.2f negative", i, s.LengthM)
		}
		if s.ThicknessM < 0 {
			t.Errorf("structure[%d] ThicknessM=%.2f negative", i, s.ThicknessM)
		}
		if s.GateCount < 0 {
			t.Errorf("structure[%d] GateCount=%d negative", i, s.GateCount)
		}
		if s.TowerCount < 0 {
			t.Errorf("structure[%d] TowerCount=%d negative", i, s.TowerCount)
		}
	}
}

func TestGenerateMilitaryStructures_DifferentIDs(t *testing.T) {
	a := newTestAnalyzer(t)
	bf1 := standardBattlefield()
	bf1.ID = 10
	bf2 := standardBattlefield()
	bf2.ID = 20

	s1 := a.GenerateMilitaryStructures(bf1)
	s2 := a.GenerateMilitaryStructures(bf2)

	for _, s := range s1 {
		if s.BattlefieldID != 10 {
			t.Errorf("s1.BattlefieldID=%d, want 10", s.BattlefieldID)
		}
	}
	for _, s := range s2 {
		if s.BattlefieldID != 20 {
			t.Errorf("s2.BattlefieldID=%d, want 20", s.BattlefieldID)
		}
	}
}
