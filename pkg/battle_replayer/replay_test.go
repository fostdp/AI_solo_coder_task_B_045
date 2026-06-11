package battle_replayer

import (
	"fmt"
	"math"
	"testing"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
	"ancient-battlefield/pkg/nlp_worker"
)

func newTestAnalyzer(t *testing.T) *ReplayAnalyzer {
	t.Helper()
	cfg := config.DefaultConfig
	return New(&cfg)
}

func standardBattlefield() models.Battlefield {
	return models.Battlefield{
		ID:           1,
		BattleName:   "长平之战",
		Era:          "春秋战国",
		BelligerentA: "秦军",
		BelligerentB: "赵军",
		TroopA:       50000,
		TroopB:       40000,
		TotalTroops:  90000,
		Lng:          112.5,
		Lat:          35.8,
	}
}

func TestNLPExtractEvents_AllEras(t *testing.T) {
	a := newTestAnalyzer(t)
	eras := []string{"春秋战国", "秦汉", "三国两晋南北朝", "隋唐五代", "宋辽金元", "明清"}

	for _, era := range eras {
		t.Run(era, func(t *testing.T) {
			bf := standardBattlefield()
			bf.Era = era
			bf.ID = 10

			events := a.NLPExtractEvents(bf)

			if len(events) < 5 || len(events) > 8 {
				t.Errorf("era %s: expected 5-8 events, got %d", era, len(events))
			}

			for i, ev := range events {
				if ev.EventType == "" {
					t.Errorf("era %s event[%d]: EventType is empty", era, i)
				}
				if ev.EventName == "" {
					t.Errorf("era %s event[%d]: EventName is empty", era, i)
				}
				if ev.Description == "" {
					t.Errorf("era %s event[%d]: Description is empty", era, i)
				}
				if ev.NLPConfidence < 0.75 || ev.NLPConfidence > 0.95 {
					t.Errorf("era %s event[%d]: NLPConfidence %.4f out of [0.75, 0.95]", era, i, ev.NLPConfidence)
				}
			}
		})
	}
}

func TestNLPExtractEvents_SortedByHourOffset(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	events := a.NLPExtractEvents(bf)

	for i := 1; i < len(events); i++ {
		if events[i].HourOffset < events[i-1].HourOffset {
			t.Errorf("events not sorted: event[%d].HourOffset=%.2f > event[%d].HourOffset=%.2f",
				i-1, events[i-1].HourOffset, i, events[i].HourOffset)
		}
	}
}

func TestNLPExtractEvents_EventOrderReassigned(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	events := a.NLPExtractEvents(bf)

	for i, ev := range events {
		if ev.EventOrder != i+1 {
			t.Errorf("event[%d]: EventOrder=%d, want %d", i, ev.EventOrder, i+1)
		}
	}
}

func TestNLPExtractEvents_CoordinatesOffset(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	events := a.NLPExtractEvents(bf)

	maxLngOffset := 0.20
	maxLatOffset := 0.16

	for i, ev := range events {
		dlng := math.Abs(ev.Lng - bf.Lng)
		dlat := math.Abs(ev.Lat - bf.Lat)
		if dlng > maxLngOffset {
			t.Errorf("event[%d]: Lng offset %.4f exceeds max %.2f", i, dlng, maxLngOffset)
		}
		if dlat > maxLatOffset {
			t.Errorf("event[%d]: Lat offset %.4f exceeds max %.2f", i, dlat, maxLatOffset)
		}
	}
}

func TestNLPExtractEvents_TurningPointsAndDecisions(t *testing.T) {
	a := newTestAnalyzer(t)

	eraTemplateChecks := map[string]struct {
		expectTurning int
		expectDecision int
	}{
		"春秋战国": {2, 1},
		"秦汉":    {3, 2},
		"三国两晋南北朝": {2, 1},
		"隋唐五代":  {2, 1},
		"宋辽金元":  {3, 2},
		"明清":    {3, 2},
	}

	for era, expect := range eraTemplateChecks {
		t.Run(era, func(t *testing.T) {
			bf := standardBattlefield()
			bf.Era = era
			bf.ID = 42

			events := a.NLPExtractEvents(bf)

			tpCount := 0
			dcCount := 0
			for _, ev := range events {
				if ev.IsTurningPoint {
					tpCount++
				}
				if ev.IsDecision {
					dcCount++
				}
			}

			if tpCount > expect.expectTurning {
				t.Errorf("era %s: turning points %d exceeds template max %d", era, tpCount, expect.expectTurning)
			}
			if dcCount > expect.expectDecision {
				t.Errorf("era %s: decisions %d exceeds template max %d", era, dcCount, expect.expectDecision)
			}

			templateTPs := 0
			templateDCs := 0
			for _, tpl := range nlp_worker.EventTemplates[era][:len(events)] {
				if tpl.Turning {
					templateTPs++
				}
				if tpl.Decision {
					templateDCs++
				}
			}
			if tpCount != templateTPs {
				t.Errorf("era %s: turning points %d, expected %d from templates", era, tpCount, templateTPs)
			}
			if dcCount != templateDCs {
				t.Errorf("era %s: decisions %d, expected %d from templates", era, dcCount, templateDCs)
			}
		})
	}
}

func TestNLPExtractEvents_TagsPreserved(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	bf.Era = "秦汉"
	bf.ID = 7

	events := a.NLPExtractEvents(bf)
	templates := nlp_worker.EventTemplates["秦汉"]

	for i, ev := range events {
		if i >= len(templates) {
			break
		}
		tpl := templates[i]
		if len(ev.Tags) != len(tpl.Tags) {
			t.Errorf("event[%d]: tag count %d != template tag count %d", i, len(ev.Tags), len(tpl.Tags))
			continue
		}
		for j, tag := range ev.Tags {
			if tag != tpl.Tags[j] {
				t.Errorf("event[%d].Tags[%d]: got %q, want %q", i, j, tag, tpl.Tags[j])
			}
		}
	}
}

func TestNLPExtractEvents_UnknownEraFallback(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	bf.Era = "未知朝代"
	bf.ID = 99

	events := a.NLPExtractEvents(bf)

	qinHanBF := standardBattlefield()
	qinHanBF.Era = "秦汉"
	qinHanBF.ID = 99
	qinHanEvents := a.NLPExtractEvents(qinHanBF)

	if len(events) != len(qinHanEvents) {
		t.Errorf("unknown era event count %d != qin han fallback count %d", len(events), len(qinHanEvents))
	}

	for i, ev := range events {
		if i >= len(qinHanEvents) {
			break
		}
		if ev.EventType != qinHanEvents[i].EventType {
			t.Errorf("event[%d]: Type %q != fallback Type %q", i, ev.EventType, qinHanEvents[i].EventType)
		}
		if ev.EventName != qinHanEvents[i].EventName {
			t.Errorf("event[%d]: Name %q != fallback Name %q", i, ev.EventName, qinHanEvents[i].EventName)
		}
	}
}

func TestBuildTimeline_Normal(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	events := a.NLPExtractEvents(bf)

	timeline := a.BuildTimeline(bf, events)

	if timeline.TotalDurationH <= 0 {
		t.Errorf("TotalDurationH=%.2f, want > 0", timeline.TotalDurationH)
	}
	if timeline.BattlefieldID != bf.ID {
		t.Errorf("BattlefieldID=%d, want %d", timeline.BattlefieldID, bf.ID)
	}
	if timeline.BattleName != bf.BattleName {
		t.Errorf("BattleName=%q, want %q", timeline.BattleName, bf.BattleName)
	}

	lastHour := events[len(events)-1].HourOffset
	expectedDuration := math.Round((lastHour+2.0)*100) / 100
	if timeline.TotalDurationH != expectedDuration {
		t.Errorf("TotalDurationH=%.2f, want %.2f", timeline.TotalDurationH, expectedDuration)
	}

	tpCount := 0
	dcCount := 0
	for _, ev := range events {
		if ev.IsTurningPoint {
			tpCount++
		}
		if ev.IsDecision {
			dcCount++
		}
	}
	if len(timeline.TurningPoints) != tpCount {
		t.Errorf("TurningPoints count=%d, want %d", len(timeline.TurningPoints), tpCount)
	}
	if len(timeline.Decisions) != dcCount {
		t.Errorf("Decisions count=%d, want %d", len(timeline.Decisions), dcCount)
	}
}

func TestBuildTimeline_EmptyEvents(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	timeline := a.BuildTimeline(bf, nil)

	if timeline.TotalDurationH != 0 {
		t.Errorf("empty events: TotalDurationH=%.2f, want 0", timeline.TotalDurationH)
	}
	if len(timeline.Events) != 0 {
		t.Errorf("empty events: Events length=%d, want 0", len(timeline.Events))
	}
	if len(timeline.TurningPoints) != 0 {
		t.Errorf("empty events: TurningPoints length=%d, want 0", len(timeline.TurningPoints))
	}
	if len(timeline.Decisions) != 0 {
		t.Errorf("empty events: Decisions length=%d, want 0", len(timeline.Decisions))
	}
}

func TestGenerateAnimationFrames_Fluency(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	events := a.NLPExtractEvents(bf)
	timeline := a.BuildTimeline(bf, events)

	fpsCases := []int{8, 12, 24}

	for _, fps := range fpsCases {
		t.Run(fmt.Sprintf("fps_%d", fps), func(t *testing.T) {
			frames := a.GenerateAnimationFrames(bf, timeline, fps)
			expectedTotal := int(math.Max(20, float64(fps)*8))

			if len(frames) != expectedTotal {
				t.Errorf("fps=%d: total frames=%d, want %d", fps, len(frames), expectedTotal)
			}

			for i := 1; i < len(frames); i++ {
				if frames[i].TimestampH < frames[i-1].TimestampH {
					t.Errorf("fps=%d: timestamps not monotonic at frame %d: %.2f < %.2f",
						fps, i, frames[i].TimestampH, frames[i-1].TimestampH)
				}
			}

			for i, f := range frames {
				if f.FrameIndex != i {
					t.Errorf("frame[%d]: FrameIndex=%d, want %d", i, f.FrameIndex, i)
				}
			}

			if len(frames) > 0 {
				totalAFirst := sideTroopTotal(t, frames[0], bf.BelligerentA)
				totalALast := sideTroopTotal(t, frames[len(frames)-1], bf.BelligerentA)
				if totalALast >= totalAFirst && bf.TroopA > 0 {
					t.Errorf("fps=%d: side A troops should decrease: first=%d, last=%d", fps, totalAFirst, totalALast)
				}

				totalBFirst := sideTroopTotal(t, frames[0], bf.BelligerentB)
				totalBLast := sideTroopTotal(t, frames[len(frames)-1], bf.BelligerentB)
				if totalBLast >= totalBFirst && bf.TroopB > 0 {
					t.Errorf("fps=%d: side B troops should decrease: first=%d, last=%d", fps, totalBFirst, totalBLast)
				}
			}
		})
	}
}

func TestGenerateAnimationFrames_FrontLinesMiddlePhase(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	events := a.NLPExtractEvents(bf)
	timeline := a.BuildTimeline(bf, events)
	frames := a.GenerateAnimationFrames(bf, timeline, 12)

	totalFrames := float64(len(frames) - 1)

	for i, f := range frames {
		t := float64(i) / totalFrames
		hasFL := len(f.FrontLines) > 0
		inMiddle := t > 0.2 && t < 0.85

		if inMiddle && !hasFL {
			t.Errorf("frame[%d] t=%.3f: expected front lines in middle phase, got none", i, t)
		}
		if !inMiddle && hasFL {
			t.Errorf("frame[%d] t=%.3f: front lines outside middle phase", i, t)
		}
	}
}

func TestGenerateAnimationFrames_ActiveEventsMatch(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	events := a.NLPExtractEvents(bf)
	timeline := a.BuildTimeline(bf, events)
	frames := a.GenerateAnimationFrames(bf, timeline, 12)

	activeFound := false
	for _, f := range frames {
		if f.ActiveEvent != nil {
			activeFound = true
			ev := f.ActiveEvent
			if f.TimestampH < ev.HourOffset || f.TimestampH >= ev.HourOffset+1.5 {
				t.Errorf("frame timestamp %.2f outside active event window [%.2f, %.2f)",
					f.TimestampH, ev.HourOffset, ev.HourOffset+1.5)
			}
			found := false
			for _, tev := range timeline.Events {
				if tev.ID == ev.ID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("active event ID %d not in timeline events", ev.ID)
			}
		}
	}
	if !activeFound && len(timeline.Events) > 0 {
		t.Error("expected at least one frame with active event")
	}
}

func TestGenerateAnimationFrames_FPSZero(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	events := a.NLPExtractEvents(bf)
	timeline := a.BuildTimeline(bf, events)

	frames := a.GenerateAnimationFrames(bf, timeline, 0)

	expectedTotal := int(math.Max(20, float64(12)*8))
	if len(frames) != expectedTotal {
		t.Errorf("fps=0: total frames=%d, want %d (default fps=12)", len(frames), expectedTotal)
	}
}

func TestGenerateAnimationFrames_FPSOne(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	events := a.NLPExtractEvents(bf)
	timeline := a.BuildTimeline(bf, events)

	frames := a.GenerateAnimationFrames(bf, timeline, 1)

	expectedTotal := int(math.Max(20, float64(1)*8))
	if len(frames) != expectedTotal {
		t.Errorf("fps=1: total frames=%d, want %d", len(frames), expectedTotal)
	}
}

func TestGenerateAnimationFrames_FPSHundred(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	events := a.NLPExtractEvents(bf)
	timeline := a.BuildTimeline(bf, events)

	frames := a.GenerateAnimationFrames(bf, timeline, 100)

	expectedTotal := int(math.Max(20, float64(100)*8))
	if len(frames) != expectedTotal {
		t.Errorf("fps=100: total frames=%d, want %d", len(frames), expectedTotal)
	}
}

func TestGenerateBattleReplay_Integration(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	fps := 12

	result := a.GenerateBattleReplay(bf, fps)

	events := a.NLPExtractEvents(bf)
	if result.NLPStats.TotalEvents != len(events) {
		t.Errorf("TotalEvents=%d, want %d", result.NLPStats.TotalEvents, len(events))
	}

	var sumConf float64
	for _, ev := range events {
		sumConf += ev.NLPConfidence
	}
	var expectedAvg float64
	if len(events) > 0 {
		expectedAvg = math.Round(sumConf/float64(len(events))*10000) / 10000
	}
	if result.NLPStats.AvgConfidence != expectedAvg {
		t.Errorf("AvgConfidence=%.4f, want %.4f", result.NLPStats.AvgConfidence, expectedAvg)
	}

	tpCount := 0
	dcCount := 0
	for _, ev := range events {
		if ev.IsTurningPoint {
			tpCount++
		}
		if ev.IsDecision {
			dcCount++
		}
	}
	if result.NLPStats.TurningPointCount != tpCount {
		t.Errorf("TurningPointCount=%d, want %d", result.NLPStats.TurningPointCount, tpCount)
	}
	if result.NLPStats.DecisionCount != dcCount {
		t.Errorf("DecisionCount=%d, want %d", result.NLPStats.DecisionCount, dcCount)
	}

	if result.Fps != fps {
		t.Errorf("Fps=%d, want %d", result.Fps, fps)
	}
	if result.TotalFrames != len(result.Frames) {
		t.Errorf("TotalFrames=%d, want %d", result.TotalFrames, len(result.Frames))
	}
}

func TestGenerateBattleReplay_CachedInGetLast(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	result := a.GenerateBattleReplay(bf, 12)

	cached := a.GetLast(bf.ID)
	if cached == nil {
		t.Fatal("GetLast returned nil, expected cached result")
	}
	if cached.Fps != result.Fps {
		t.Errorf("cached Fps=%d, want %d", cached.Fps, result.Fps)
	}
	if cached.TotalFrames != result.TotalFrames {
		t.Errorf("cached TotalFrames=%d, want %d", cached.TotalFrames, result.TotalFrames)
	}
	if cached.NLPStats.TotalEvents != result.NLPStats.TotalEvents {
		t.Errorf("cached TotalEvents=%d, want %d", cached.NLPStats.TotalEvents, result.NLPStats.TotalEvents)
	}

	missing := a.GetLast(99999)
	if missing != nil {
		t.Error("GetLast for unknown ID should return nil")
	}
}

func TestDeterminism(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	result1 := a.GenerateBattleReplay(bf, 12)
	result2 := a.GenerateBattleReplay(bf, 12)

	if result1.NLPStats.TotalEvents != result2.NLPStats.TotalEvents {
		t.Errorf("non-deterministic TotalEvents: %d vs %d", result1.NLPStats.TotalEvents, result2.NLPStats.TotalEvents)
	}
	if result1.NLPStats.AvgConfidence != result2.NLPStats.AvgConfidence {
		t.Errorf("non-deterministic AvgConfidence: %.4f vs %.4f", result1.NLPStats.AvgConfidence, result2.NLPStats.AvgConfidence)
	}
	if result1.TotalFrames != result2.TotalFrames {
		t.Errorf("non-deterministic TotalFrames: %d vs %d", result1.TotalFrames, result2.TotalFrames)
	}
	if len(result1.Frames) != len(result2.Frames) {
		t.Errorf("non-deterministic frame count: %d vs %d", len(result1.Frames), len(result2.Frames))
	}

	events1 := a.NLPExtractEvents(bf)
	events2 := a.NLPExtractEvents(bf)
	for i := range events1 {
		if i >= len(events2) {
			break
		}
		if events1[i].ID != events2[i].ID {
			t.Errorf("non-deterministic event[%d].ID: %d vs %d", i, events1[i].ID, events2[i].ID)
		}
		if events1[i].HourOffset != events2[i].HourOffset {
			t.Errorf("non-deterministic event[%d].HourOffset: %.4f vs %.4f", i, events1[i].HourOffset, events2[i].HourOffset)
		}
		if events1[i].NLPConfidence != events2[i].NLPConfidence {
			t.Errorf("non-deterministic event[%d].NLPConfidence: %.4f vs %.4f", i, events1[i].NLPConfidence, events2[i].NLPConfidence)
		}
	}

	for i := range result1.Frames {
		if i >= len(result2.Frames) {
			break
		}
		f1 := result1.Frames[i]
		f2 := result2.Frames[i]
		if f1.TimestampH != f2.TimestampH {
			t.Errorf("non-deterministic frame[%d].TimestampH: %.4f vs %.4f", i, f1.TimestampH, f2.TimestampH)
		}
		if len(f1.TroopPositions) != len(f2.TroopPositions) {
			t.Errorf("non-deterministic frame[%d] troop count: %d vs %d", i, len(f1.TroopPositions), len(f2.TroopPositions))
		}
	}
}

func TestZeroTroopsBattlefield(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := models.Battlefield{
		ID:           200,
		BattleName:   "空营之战",
		Era:          "秦汉",
		BelligerentA: "甲军",
		BelligerentB: "乙军",
		TroopA:       0,
		TroopB:       0,
		TotalTroops:  0,
		Lng:          110.0,
		Lat:          30.0,
	}

	events := a.NLPExtractEvents(bf)
	if len(events) < 5 {
		t.Errorf("zero troops: expected at least 5 events, got %d", len(events))
	}
	for i, ev := range events {
		if ev.TroopCount != 0 {
			t.Errorf("zero troops event[%d]: TroopCount=%d, want 0", i, ev.TroopCount)
		}
		if ev.Casualties != 0 {
			t.Errorf("zero troops event[%d]: Casualties=%d, want 0", i, ev.Casualties)
		}
	}

	timeline := a.BuildTimeline(bf, events)
	if timeline.TotalDurationH <= 0 {
		t.Errorf("zero troops: TotalDurationH=%.2f, want > 0", timeline.TotalDurationH)
	}

	result := a.GenerateBattleReplay(bf, 12)
	for i, f := range result.Frames {
		for _, pos := range f.TroopPositions {
			if pos.TroopCount != 0 {
				t.Errorf("zero troops frame[%d] unit %s: TroopCount=%d, want 0", i, pos.UnitName, pos.TroopCount)
			}
		}
	}
}

func TestNegativeTroopsBattlefield(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := models.Battlefield{
		ID:           201,
		BattleName:   "负数之战",
		Era:          "明清",
		BelligerentA: "甲军",
		BelligerentB: "乙军",
		TroopA:       -1000,
		TroopB:       -500,
		TotalTroops:  -1500,
		Lng:          116.0,
		Lat:          39.0,
	}

	result := a.GenerateBattleReplay(bf, 12)

	if result.NLPStats.TotalEvents < 5 {
		t.Errorf("negative troops: expected at least 5 events, got %d", result.NLPStats.TotalEvents)
	}
	if result.TotalFrames < 20 {
		t.Errorf("negative troops: expected at least 20 frames, got %d", result.TotalFrames)
	}
}

func TestExtremeCoordinates_SouthWest(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := models.Battlefield{
		ID:           300,
		BattleName:   "西南极地之战",
		Era:          "隋唐五代",
		BelligerentA: "甲军",
		BelligerentB: "乙军",
		TroopA:       30000,
		TroopB:       20000,
		TotalTroops:  50000,
		Lng:          73.0,
		Lat:          18.0,
	}

	events := a.NLPExtractEvents(bf)
	if len(events) < 5 {
		t.Errorf("extreme SW: expected at least 5 events, got %d", len(events))
	}
	for i, ev := range events {
		if ev.Lng < 72.5 || ev.Lng > 73.5 {
			t.Errorf("extreme SW event[%d]: Lng=%.4f too far from base 73.0", i, ev.Lng)
		}
		if ev.Lat < 17.5 || ev.Lat > 18.5 {
			t.Errorf("extreme SW event[%d]: Lat=%.4f too far from base 18.0", i, ev.Lat)
		}
	}

	result := a.GenerateBattleReplay(bf, 12)
	if result.TotalFrames < 20 {
		t.Errorf("extreme SW: TotalFrames=%d, want >= 20", result.TotalFrames)
	}
}

func TestExtremeCoordinates_NorthEast(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := models.Battlefield{
		ID:           301,
		BattleName:   "东北极地之战",
		Era:          "宋辽金元",
		BelligerentA: "甲军",
		BelligerentB: "乙军",
		TroopA:       40000,
		TroopB:       35000,
		TotalTroops:  75000,
		Lng:          135.0,
		Lat:          54.0,
	}

	events := a.NLPExtractEvents(bf)
	if len(events) < 5 {
		t.Errorf("extreme NE: expected at least 5 events, got %d", len(events))
	}
	for i, ev := range events {
		if ev.Lng < 134.5 || ev.Lng > 135.5 {
			t.Errorf("extreme NE event[%d]: Lng=%.4f too far from base 135.0", i, ev.Lng)
		}
		if ev.Lat < 53.5 || ev.Lat > 54.5 {
			t.Errorf("extreme NE event[%d]: Lat=%.4f too far from base 54.0", i, ev.Lat)
		}
	}

	result := a.GenerateBattleReplay(bf, 12)
	if result.TotalFrames < 20 {
		t.Errorf("extreme NE: TotalFrames=%d, want >= 20", result.TotalFrames)
	}
}

func TestGenerateBattleReplay_NegativeFPS(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	result := a.GenerateBattleReplay(bf, -5)

	if result.Fps != 12 {
		t.Errorf("fps=-5: result.Fps=%d, want 12 (default)", result.Fps)
	}
	expectedTotal := int(math.Max(20, float64(12)*8))
	if result.TotalFrames != expectedTotal {
		t.Errorf("fps=-5: TotalFrames=%d, want %d", result.TotalFrames, expectedTotal)
	}
}

func TestNLPExtractEvents_BelligerentAlternates(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	events := a.NLPExtractEvents(bf)

	for i, ev := range events {
		expected := []string{bf.BelligerentA, bf.BelligerentB}[i%2]
		if ev.Belligerent != expected {
			t.Errorf("event[%d]: Belligerent=%q, want %q", i, ev.Belligerent, expected)
		}
	}
}

func TestNLPExtractEvents_ExtractedFrom(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()

	events := a.NLPExtractEvents(bf)

	for i, ev := range events {
		expected := "模拟战史文本: " + bf.BattleName
		if ev.ExtractedFrom != expected {
			t.Errorf("event[%d]: ExtractedFrom=%q, want %q", i, ev.ExtractedFrom, expected)
		}
	}
}

func TestNLPExtractEvents_BattlefieldID(t *testing.T) {
	a := newTestAnalyzer(t)
	bf := standardBattlefield()
	bf.ID = 42

	events := a.NLPExtractEvents(bf)

	for i, ev := range events {
		if ev.BattlefieldID != bf.ID {
			t.Errorf("event[%d]: BattlefieldID=%d, want %d", i, ev.BattlefieldID, bf.ID)
		}
	}
}

func sideTroopTotal(t *testing.T, frame models.AnimationFrame, belligerent string) int {
	t.Helper()
	total := 0
	for _, pos := range frame.TroopPositions {
		if pos.Belligerent == belligerent {
			total += pos.TroopCount
		}
	}
	return total
}
