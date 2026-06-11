package doctrine_evolution

import (
	"strings"
	"testing"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
)

func newTestAnalyzer(t *testing.T) *DoctrineAnalyzer {
	t.Helper()
	cfg := config.DefaultConfig
	return New(&cfg)
}

var expectedEras = []struct {
	era            string
	start, end     int
	doctrine       string
}{
	{"春秋战国", -770, -221, "车战为主"},
	{"秦汉", -221, 220, "骑兵崛起"},
	{"三国两晋南北朝", 220, 589, "骑步协同"},
	{"隋唐五代", 581, 960, "步骑协同"},
	{"宋辽金元", 960, 1368, "以步制骑/骑兵巅峰"},
	{"明清", 1368, 1912, "火器时代"},
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func generateBattlefieldsPerEra(t *testing.T, perEra int, baseTroops int) []models.Battlefield {
	t.Helper()
	bfs := make([]models.Battlefield, 0, perEra*len(expectedEras))
	id := 1
	terrains := []string{"平原", "河谷", "关隘", "山地"}
	lngs := []float64{110.0, 112.5, 115.0, 108.0, 118.0, 116.0}
	lats := []float64{34.0, 36.0, 38.0, 32.0, 39.0, 40.0}

	for ei, ed := range expectedEras {
		yearSpan := float64(ed.end - ed.start)
		for i := 0; i < perEra; i++ {
			yearFrac := float64(i) / float64(perEra)
			year := ed.start + int(yearSpan*yearFrac)
			if year == ed.start && i > 0 {
				year++
			}
			elevation := 100.0 + float64(ei)*150.0 + float64(i%3)*50.0
			troops := baseTroops + ei*baseTroops*2 + i*1000
			bfs = append(bfs, models.Battlefield{
				ID:              id,
				BattleName:      "模拟战役" + itoa(id),
				Era:             ed.era,
				BattleYear:      year,
				BelligerentA:    "甲方",
				BelligerentB:    "乙方",
				TroopA:          troops / 2,
				TroopB:          troops / 2,
				TotalTroops:     troops,
				TerrainType:     terrains[(ei+i)%len(terrains)],
				Lng:             lngs[ei%len(lngs)] + float64(i%3)*0.5,
				Lat:             lats[ei%len(lats)] + float64(i%2)*0.3,
				Elevation:       elevation,
				DistanceToRiver: 2.0 + float64(i%5)*0.8,
				DistanceToRoad:  3.0 + float64(ei)*0.5,
			})
			id++
		}
	}
	return bfs
}

func TestNew_Function(t *testing.T) {
	cfg := config.DefaultConfig
	a := New(&cfg)
	if a == nil {
		t.Fatalf("New returned nil")
	}
	if a.cfg != &cfg {
		t.Errorf("cfg pointer mismatch")
	}
	if a.lastResult != nil {
		t.Errorf("lastResult should initially be nil")
	}
}

func TestComputeEraProfiles_Accuracy(t *testing.T) {
	a := newTestAnalyzer(t)
	perEra := 10
	bfs := generateBattlefieldsPerEra(t, perEra, 10000)

	profiles := a.ComputeEraProfiles(bfs)

	if len(profiles) != len(expectedEras) {
		t.Fatalf("expected %d profiles (one per era), got %d", len(expectedEras), len(profiles))
	}

	for i, p := range profiles {
		exp := expectedEras[i]
		if p.Era != exp.era {
			t.Errorf("profile[%d].Era=%q, want %q", i, p.Era, exp.era)
		}
		if p.YearRange[0] != exp.start || p.YearRange[1] != exp.end {
			t.Errorf("profile[%d] %q YearRange=[%d,%d], want [%d,%d]",
				i, p.Era, p.YearRange[0], p.YearRange[1], exp.start, exp.end)
		}
		if p.DoctrineTag != exp.doctrine {
			t.Errorf("profile[%d] %q DoctrineTag=%q, want %q",
				i, p.Era, p.DoctrineTag, exp.doctrine)
		}
		if p.BattleCount != perEra {
			t.Errorf("profile[%d] %q BattleCount=%d, want %d",
				i, p.Era, p.BattleCount, perEra)
		}
		if p.Characteristic == "" {
			t.Errorf("profile[%d] %q Characteristic is empty", i, p.Era)
		}
		if p.DominantTerrain == "" {
			t.Errorf("profile[%d] %q DominantTerrain is empty", i, p.Era)
		}

		if p.TerrainDist == nil {
			t.Errorf("profile[%d] %q TerrainDist is nil", i, p.Era)
			continue
		}
		sum := 0.0
		for _, v := range p.TerrainDist {
			sum += v
		}
		if sum < 0.99 || sum > 1.01 {
			t.Errorf("profile[%d] %q TerrainDist sums to %.4f, want ~1.0",
				i, p.Era, sum)
		}
	}
}

func TestComputeEraProfiles_Boundary(t *testing.T) {
	a := newTestAnalyzer(t)

	empty := a.ComputeEraProfiles(nil)
	if len(empty) != 0 {
		t.Errorf("empty input: expected 0 profiles, got %d", len(empty))
	}

	bf := generateBattlefieldsPerEra(t, 1, 10000)[:1]
	single := a.ComputeEraProfiles(bf)
	if len(single) != 1 {
		t.Errorf("single battlefield: expected 1 profile, got %d", len(single))
	}
	if single[0].BattleCount != 1 {
		t.Errorf("single battlefield BattleCount=%d, want 1", single[0].BattleCount)
	}
}

func TestComputeEraProfiles_Sorted(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 5, 10000)
	profiles := a.ComputeEraProfiles(bfs)

	for i := 1; i < len(profiles); i++ {
		if profiles[i].YearRange[0] < profiles[i-1].YearRange[0] {
			t.Errorf("profiles not sorted by YearRange[0] asc: [%d]=%d > [%d]=%d",
				i-1, profiles[i-1].YearRange[0], i, profiles[i].YearRange[0])
		}
	}
}

func TestComputeEraProfiles_AvgTroopsAndDistances(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := models.Battlefield{
		ID:              1,
		BattleName:      "test",
		Era:             "秦汉",
		BattleYear:      -100,
		BelligerentA:    "A",
		BelligerentB:    "B",
		TroopA:          15000,
		TroopB:          15000,
		TotalTroops:     30000,
		TerrainType:     "平原",
		Lng:             110,
		Lat:             35,
		Elevation:       500,
		DistanceToRiver: 4.5,
		DistanceToRoad:  3.0,
	}

	profiles := a.ComputeEraProfiles([]models.Battlefield{bf})
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	p := profiles[0]
	if p.AvgTroops != 30000.0 {
		t.Errorf("AvgTroops=%.2f, want 30000.0", p.AvgTroops)
	}
	if p.AvgElevation != 500.0 {
		t.Errorf("AvgElevation=%.2f, want 500.0", p.AvgElevation)
	}
	if p.AvgDistToRiver != 4.5 {
		t.Errorf("AvgDistToRiver=%.2f, want 4.5", p.AvgDistToRiver)
	}
	if p.AvgDistToRoad != 3.0 {
		t.Errorf("AvgDistToRoad=%.2f, want 3.0", p.AvgDistToRoad)
	}
}

func TestComputeChangePoints_Accuracy(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 8, 10000)
	profiles := a.ComputeEraProfiles(bfs)

	cps := a.ComputeChangePoints(profiles, bfs)

	if len(cps) != 5 {
		t.Fatalf("expected 5 change points (6 eras → 5 boundaries), got %d", len(cps))
	}

	expectedYears := []int{-221, 220, 581, 960, 1368}
	for i, cp := range cps {
		if cp.ID == 0 {
			t.Errorf("cp[%d]: ID is zero", i)
		}
		if cp.Year != expectedYears[i] {
			t.Errorf("cp[%d]: Year=%d, want %d", i, cp.Year, expectedYears[i])
		}
		if cp.EraBoundary == "" {
			t.Errorf("cp[%d]: EraBoundary is empty", i)
		}
		if cp.BeforeDoctrine == "" || cp.AfterDoctrine == "" {
			t.Errorf("cp[%d]: doctrine empty (before=%q after=%q)",
				i, cp.BeforeDoctrine, cp.AfterDoctrine)
		}
		if cp.Confidence < 0.5 || cp.Confidence > 1.01 {
			t.Errorf("cp[%d] %q: Confidence=%.4f out of [0.5,1.0]",
				i, cp.EraBoundary, cp.Confidence)
		}
		if cp.ChangeMagnitude < 0 || cp.ChangeMagnitude > 1.01 {
			t.Errorf("cp[%d] %q: ChangeMagnitude=%.4f out of [0,1]",
				i, cp.EraBoundary, cp.ChangeMagnitude)
		}
		if len(cp.KeyFeatures) == 0 {
			t.Errorf("cp[%d] %q: KeyFeatures is empty", i, cp.EraBoundary)
		}
		if len(cp.TriggerEvents) == 0 {
			t.Errorf("cp[%d] %q: TriggerEvents is empty", i, cp.EraBoundary)
		}
	}
}

func TestComputeChangePoints_DoctrineTransitions(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 5, 10000)
	profiles := a.ComputeEraProfiles(bfs)
	cps := a.ComputeChangePoints(profiles, bfs)

	expectedTransitions := []struct {
		before, after string
	}{
		{"车战为主", "骑兵崛起"},
		{"骑兵崛起", "骑步协同"},
		{"骑步协同", "步骑协同"},
		{"步骑协同", "以步制骑/骑兵巅峰"},
		{"以步制骑/骑兵巅峰", "火器时代"},
	}

	for i, exp := range expectedTransitions {
		if i >= len(cps) {
			break
		}
		if cps[i].BeforeDoctrine != exp.before {
			t.Errorf("cp[%d] BeforeDoctrine=%q, want %q", i, cps[i].BeforeDoctrine, exp.before)
		}
		if cps[i].AfterDoctrine != exp.after {
			t.Errorf("cp[%d] AfterDoctrine=%q, want %q", i, cps[i].AfterDoctrine, exp.after)
		}
	}
}

func TestDetectChangePoints_Indirect(t *testing.T) {
	a := newTestAnalyzer(t)

	jumpBfs := make([]models.Battlefield, 0)
	for i := 0; i < 20; i++ {
		bf := models.Battlefield{
			ID:          i + 1,
			BattleName:  "j" + itoa(i),
			Era:         "春秋战国",
			BattleYear:  -700 + i*5,
			TotalTroops: 10000,
			TerrainType: "平原",
			Elevation:   100,
		}
		if i >= 10 {
			bf.Elevation = 800
			bf.Era = "秦汉"
			bf.BattleYear = -100 + (i-10)*5
		}
		jumpBfs = append(jumpBfs, bf)
	}

	profiles := a.ComputeEraProfiles(jumpBfs)
	cps := a.ComputeChangePoints(profiles, jumpBfs)

	if len(cps) < 1 {
		t.Errorf("expected at least 1 change point for elevation jump, got %d", len(cps))
	}
}

func TestComputeTimeAnimation_FrameCount(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 5, 10000)
	profiles := a.ComputeEraProfiles(bfs)

	anim := a.ComputeTimeAnimation(profiles, bfs)

	if len(anim) != 24 {
		t.Fatalf("expected 24 animation frames, got %d", len(anim))
	}

	for i, fr := range anim {
		if fr.Era == "" {
			t.Errorf("frame[%d]: Era is empty", i)
		}
		if fr.DoctrineTag == "" {
			t.Errorf("frame[%d]: DoctrineTag is empty", i)
		}
		if i > 0 {
			if fr.Year <= anim[i-1].Year {
				t.Errorf("frame[%d] Year=%d not monotonic (prev=%d)",
					i, fr.Year, anim[i-1].Year)
			}
		}
		if len(fr.HotspotCenters) < 1 {
			t.Errorf("frame[%d]: %d hotspots, want >=1", i, len(fr.HotspotCenters))
		}
		if len(fr.HeatmapData) < 30 {
			t.Errorf("frame[%d]: %d heatmap samples, want >=30", i, len(fr.HeatmapData))
		}
		if len(fr.Features) == 0 {
			t.Errorf("frame[%d]: Features is empty", i)
		}
		for fk, fv := range fr.Features {
			if fv < 0 {
				t.Errorf("frame[%d] feature %q=%.4f negative", i, fk, fv)
			}
		}
	}
}

func TestComputeTimeAnimation_CoordinatesValid(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 5, 10000)
	profiles := a.ComputeEraProfiles(bfs)
	anim := a.ComputeTimeAnimation(profiles, bfs)

	for i, fr := range anim {
		for hi, hs := range fr.HotspotCenters {
			lng, lat := hs[0], hs[1]
			if lng < 70 || lng > 140 || lat < 10 || lat > 60 {
				t.Errorf("frame[%d] hotspot[%d]: (%.4f,%.4f) out of China range",
					i, hi, lng, lat)
			}
		}
		for di, d := range fr.HeatmapData {
			if d.Lng < 70 || d.Lng > 140 || d.Lat < 10 || d.Lat > 60 {
				t.Errorf("frame[%d] heat[%d]: (%.4f,%.4f) out of China range",
					i, di, d.Lng, d.Lat)
			}
			if d.Value < 0 {
				t.Errorf("frame[%d] heat[%d]: Value=%.4f negative", i, di, d.Value)
			}
		}
	}
}

func TestComputeTimeSeries_Smoothness(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 20, 10000)

	ts := a.ComputeTimeSeries(bfs)

	if len(ts) == 0 {
		t.Fatalf("time series is empty")
	}

	for i := 1; i < len(ts); i++ {
		if ts[i].Year < ts[i-1].Year {
			t.Errorf("ts[%d] Year=%d < previous=%d, not monotonic",
				i, ts[i].Year, ts[i-1].Year)
		}
	}

	first := ts[0].Year
	last := ts[len(ts)-1].Year
	if first > -600 {
		t.Errorf("first year %d should be earlier than -600", first)
	}
	if last < 1500 {
		t.Errorf("last year %d should be later than 1500", last)
	}
}

func TestComputeTimeSeries_EmptyInput(t *testing.T) {
	a := newTestAnalyzer(t)
	ts := a.ComputeTimeSeries(nil)
	if ts != nil {
		t.Errorf("empty input should return nil, got %d points", len(ts))
	}
}

func TestAnalyzeEvolution_Integration(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 6, 10000)

	result := a.AnalyzeEvolution(bfs)

	if len(result.Profiles) != 6 {
		t.Errorf("expected 6 Profiles, got %d", len(result.Profiles))
	}
	if len(result.ChangePoints) != 5 {
		t.Errorf("expected 5 ChangePoints, got %d", len(result.ChangePoints))
	}
	if len(result.TimeAnimation) != 24 {
		t.Errorf("expected 24 TimeAnimation frames, got %d", len(result.TimeAnimation))
	}
	if len(result.TimeSeries) == 0 {
		t.Errorf("TimeSeries should not be empty")
	}
	if len(result.SummaryTrends) == 0 {
		t.Errorf("SummaryTrends should not be empty")
	}

	cached := a.GetLast()
	if cached == nil {
		t.Fatalf("GetLast returned nil after AnalyzeEvolution")
	}
	if len(cached.Profiles) != len(result.Profiles) {
		t.Errorf("cached Profiles count mismatch")
	}
}

func TestAnalyzeEvolution_SummaryTrendsKeywords(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 5, 10000)
	result := a.AnalyzeEvolution(bfs)

	allText := ""
	for k, v := range result.SummaryTrends {
		allText += k + " " + v + " "
	}

	mustHave := []string{"兵力", "海拔", "骑兵", "火器"}
	foundCount := 0
	for _, kw := range mustHave {
		if strings.Contains(allText, kw) {
			foundCount++
		}
	}
	if foundCount < 3 {
		t.Errorf("SummaryTrends expected to contain at least 3 of %v, found %d. All text: %s",
			mustHave, foundCount, allText)
	}
}

func TestAnalyzeEvolution_EmptyInput(t *testing.T) {
	a := newTestAnalyzer(t)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AnalyzeEvolution(nil) panicked: %v", r)
		}
	}()
	result := a.AnalyzeEvolution(nil)
	_ = result
}

func TestCaching_GetLastBeforeCall(t *testing.T) {
	a := newTestAnalyzer(t)
	got := a.GetLast()
	if got != nil {
		t.Errorf("GetLast before any AnalyzeEvolution should be nil, got %+v", got)
	}
}

func TestCaching_GetLastReturnsSamePointer(t *testing.T) {
	a := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 3, 10000)

	first := a.AnalyzeEvolution(bfs)
	cached := a.GetLast()

	if len(first.Profiles) != len(cached.Profiles) {
		t.Errorf("cached Profiles count differs")
	}
	for i := range first.Profiles {
		if first.Profiles[i].Era != cached.Profiles[i].Era {
			t.Errorf("cached Profiles[%d].Era differs: %q vs %q",
				i, first.Profiles[i].Era, cached.Profiles[i].Era)
		}
	}
}

func TestDeterminism(t *testing.T) {
	a1 := newTestAnalyzer(t)
	a2 := newTestAnalyzer(t)
	bfs := generateBattlefieldsPerEra(t, 5, 10000)

	r1 := a1.AnalyzeEvolution(bfs)
	r2 := a2.AnalyzeEvolution(bfs)

	if len(r1.Profiles) != len(r2.Profiles) {
		t.Fatalf("Profiles count differs: %d vs %d", len(r1.Profiles), len(r2.Profiles))
	}
	for i := range r1.Profiles {
		if r1.Profiles[i].Era != r2.Profiles[i].Era {
			t.Errorf("Profiles[%d].Era differs: %q vs %q",
				i, r1.Profiles[i].Era, r2.Profiles[i].Era)
		}
		if r1.Profiles[i].BattleCount != r2.Profiles[i].BattleCount {
			t.Errorf("Profiles[%d].BattleCount differs: %d vs %d",
				i, r1.Profiles[i].BattleCount, r2.Profiles[i].BattleCount)
		}
	}

	if len(r1.ChangePoints) != len(r2.ChangePoints) {
		t.Fatalf("ChangePoints count differs: %d vs %d",
			len(r1.ChangePoints), len(r2.ChangePoints))
	}
	for i := range r1.ChangePoints {
		if r1.ChangePoints[i].Year != r2.ChangePoints[i].Year {
			t.Errorf("ChangePoints[%d].Year differs: %d vs %d",
				i, r1.ChangePoints[i].Year, r2.ChangePoints[i].Year)
		}
	}

	if len(r1.TimeAnimation) != len(r2.TimeAnimation) {
		t.Fatalf("TimeAnimation count differs: %d vs %d",
			len(r1.TimeAnimation), len(r2.TimeAnimation))
	}
	for i := range r1.TimeAnimation {
		if r1.TimeAnimation[i].Year != r2.TimeAnimation[i].Year {
			t.Errorf("TimeAnimation[%d].Year differs: %d vs %d",
				i, r1.TimeAnimation[i].Year, r2.TimeAnimation[i].Year)
		}
	}

	if len(r1.TimeSeries) != len(r2.TimeSeries) {
		t.Fatalf("TimeSeries count differs: %d vs %d",
			len(r1.TimeSeries), len(r2.TimeSeries))
	}
}
