package nlp_worker

import (
	"math"
	"testing"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
)

func standardBF() models.Battlefield {
	return models.Battlefield{
		ID: 1, BattleName: "长平之战", Era: "春秋战国", Dynasty: "战国",
		BattleYear: -260, BelligerentA: "秦军", BelligerentB: "赵军",
		TroopA: 50000, TroopB: 40000, TotalTroops: 90000,
		Lng: 112.5, Lat: 35.8, Elevation: 1200,
	}
}

func TestNew_ReturnsNonNull(t *testing.T) {
	w := New(&config.ModelConfig{})
	if w == nil {
		t.Fatal("New returned nil")
	}
}

func TestEventTemplates_AllEras(t *testing.T) {
	expected := []string{"春秋战国", "秦汉", "三国两晋南北朝", "隋唐五代", "宋辽金元", "明清"}
	for _, era := range expected {
		if _, ok := EventTemplates[era]; !ok {
			t.Errorf("EventTemplates missing era key: %s", era)
		}
	}
	if len(EventTemplates) != len(expected) {
		t.Errorf("EventTemplates has %d keys, want %d", len(EventTemplates), len(expected))
	}
}

func TestEventTemplates_EraEventCount(t *testing.T) {
	for era, templates := range EventTemplates {
		if len(templates) < 5 || len(templates) > 8 {
			t.Errorf("era %s has %d event templates, want 5-8", era, len(templates))
		}
	}
}

func TestEventTemplates_Types(t *testing.T) {
	for era, templates := range EventTemplates {
		for i, tpl := range templates {
			if tpl.Type == "" {
				t.Errorf("era %s template[%d] has empty Type", era, i)
			}
			if tpl.Name == "" {
				t.Errorf("era %s template[%d] has empty Name", era, i)
			}
			if tpl.Description == "" {
				t.Errorf("era %s template[%d] has empty Description", era, i)
			}
		}
	}
}

func TestDoctrineEventMap_Keys(t *testing.T) {
	expected := []string{"车战为主", "骑兵崛起", "步骑协同", "火器时代"}
	for _, key := range expected {
		if _, ok := DoctrineEventMap[key]; !ok {
			t.Errorf("DoctrineEventMap missing key: %s", key)
		}
	}
	if len(DoctrineEventMap) != len(expected) {
		t.Errorf("DoctrineEventMap has %d keys, want %d", len(DoctrineEventMap), len(expected))
	}
}

func TestExtractEvents_StandardBattlefield(t *testing.T) {
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(standardBF())
	if len(events) < 5 || len(events) > 8 {
		t.Errorf("got %d events, want 5-8", len(events))
	}
}

func TestExtractEvents_EventFields(t *testing.T) {
	bf := standardBF()
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(bf)
	belligerents := map[string]bool{bf.BelligerentA: true, bf.BelligerentB: true}
	for i, ev := range events {
		if !belligerents[ev.Belligerent] {
			t.Errorf("event[%d] Belligerent=%q not in belligerents", i, ev.Belligerent)
		}
		if ev.NLPConfidence < 0 || ev.NLPConfidence > 1 {
			t.Errorf("event[%d] NLPConfidence=%f not in [0,1]", i, ev.NLPConfidence)
		}
		if ev.TroopCount < 0 {
			t.Errorf("event[%d] TroopCount=%d < 0", i, ev.TroopCount)
		}
	}
}

func TestExtractEvents_HourOffsetSorted(t *testing.T) {
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(standardBF())
	for i := 1; i < len(events); i++ {
		if events[i].HourOffset < events[i-1].HourOffset {
			t.Errorf("events[%d].HourOffset=%f < events[%d].HourOffset=%f",
				i, events[i].HourOffset, i-1, events[i-1].HourOffset)
		}
	}
}

func TestExtractEvents_EventOrderSequential(t *testing.T) {
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(standardBF())
	for i, ev := range events {
		if ev.EventOrder != i+1 {
			t.Errorf("events[%d].EventOrder=%d, want %d", i, ev.EventOrder, i+1)
		}
	}
}

func TestExtractEvents_TurningPointAndDecision(t *testing.T) {
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(standardBF())
	hasTurning := false
	hasDecision := false
	for _, ev := range events {
		if ev.IsTurningPoint {
			hasTurning = true
		}
		if ev.IsDecision {
			hasDecision = true
		}
	}
	if !hasTurning {
		t.Error("no turning point found in events")
	}
	if !hasDecision {
		t.Error("no decision point found in events")
	}
}

func TestExtractEvents_DeployEventPresent(t *testing.T) {
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(standardBF())
	hasDeploy := false
	for _, ev := range events {
		if ev.EventType == "部署" {
			hasDeploy = true
		}
	}
	if !hasDeploy {
		t.Error("no deploy event found in events")
	}
}

func TestExtractEvents_KGComplemented(t *testing.T) {
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(standardBF())
	for _, ev := range events {
		_ = ev.KGComplemented
	}
}

func TestExtractEvents_ExpertVerifiedAll(t *testing.T) {
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(standardBF())
	for i, ev := range events {
		if !ev.ExpertVerified {
			t.Errorf("events[%d] ExpertVerified=false, want true", i)
		}
	}
}

func TestExtractEvents_UnknownEra(t *testing.T) {
	bf := standardBF()
	bf.Era = "未知朝代"
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(bf)
	if len(events) == 0 {
		t.Error("expected non-empty events for unknown era")
	}
	for _, ev := range events {
		if ev.BattlefieldID != bf.ID {
			t.Errorf("event BattlefieldID=%d, want %d", ev.BattlefieldID, bf.ID)
		}
	}
}

func TestExtractEvents_ZeroTroops(t *testing.T) {
	bf := standardBF()
	bf.TotalTroops = 0
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(bf)
	if len(events) == 0 {
		t.Error("expected non-empty events with zero troops")
	}
}

func TestExtractEvents_NegativeTroops(t *testing.T) {
	bf := standardBF()
	bf.TotalTroops = -1000
	w := New(&config.DefaultConfig)
	events := w.ExtractEvents(bf)
	if len(events) == 0 {
		t.Error("expected non-empty events with negative troops")
	}
}

func TestExtractEvents_Determinism(t *testing.T) {
	w := New(&config.DefaultConfig)
	bf := standardBF()
	events1 := w.ExtractEvents(bf)
	events2 := w.ExtractEvents(bf)
	if len(events1) != len(events2) {
		t.Fatalf("determinism check: len %d vs %d", len(events1), len(events2))
	}
	for i := range events1 {
		if events1[i].EventType != events2[i].EventType {
			t.Errorf("events[%d] EventType mismatch: %q vs %q", i, events1[i].EventType, events2[i].EventType)
		}
		if math.Abs(events1[i].HourOffset-events2[i].HourOffset) > 1e-9 {
			t.Errorf("events[%d] HourOffset mismatch: %f vs %f", i, events1[i].HourOffset, events2[i].HourOffset)
		}
	}
}

func TestKgComplement_AddsMissingTypes(t *testing.T) {
	bf := standardBF()
	events := []models.BattleEvent{
		{EventType: "进军", BattlefieldID: bf.ID, HourOffset: 0},
		{EventType: "撤退", BattlefieldID: bf.ID, HourOffset: 5},
	}
	templates := EventTemplates[bf.Era]
	result := KgComplement(bf, events, templates)
	hasDeploy := false
	hasBattle := false
	hasDecisive := false
	for _, ev := range result {
		if ev.EventType == "部署" {
			hasDeploy = true
		}
		if ev.EventType == "交战" {
			hasBattle = true
		}
		if ev.EventType == "决战" {
			hasDecisive = true
		}
	}
	if !hasDeploy {
		t.Error("KgComplement did not add 部署 event")
	}
	if !hasBattle {
		t.Error("KgComplement did not add 交战 event")
	}
	if !hasDecisive {
		t.Error("KgComplement did not add 决战 event")
	}
}

func TestExpertValidate_FixesNegatives(t *testing.T) {
	bf := standardBF()
	events := []models.BattleEvent{
		{EventType: "部署", TroopCount: -100, Casualties: -50, HourOffset: 1, IsTurningPoint: true, IsDecision: true, Tags: []string{}},
		{EventType: "交战", TroopCount: -200, Casualties: -80, HourOffset: 2, Tags: []string{}},
	}
	result := ExpertValidate(bf, events)
	for i, ev := range result {
		if ev.TroopCount < 0 {
			t.Errorf("result[%d] TroopCount=%d, want >=0", i, ev.TroopCount)
		}
		if ev.Casualties < 0 {
			t.Errorf("result[%d] Casualties=%d, want >=0", i, ev.Casualties)
		}
	}
}

func TestExpertValidate_AddsTurningPoint(t *testing.T) {
	bf := standardBF()
	events := []models.BattleEvent{
		{EventType: "进军", TroopCount: 100, Casualties: 10, HourOffset: 1, IsTurningPoint: false, IsDecision: false, Tags: []string{}},
		{EventType: "交战", TroopCount: 200, Casualties: 20, HourOffset: 3, IsTurningPoint: false, IsDecision: false, Tags: []string{}},
		{EventType: "撤退", TroopCount: 50, Casualties: 5, HourOffset: 5, IsTurningPoint: false, IsDecision: false, Tags: []string{}},
	}
	result := ExpertValidate(bf, events)
	hasTurning := false
	hasDecision := false
	for _, ev := range result {
		if ev.IsTurningPoint {
			hasTurning = true
		}
		if ev.IsDecision {
			hasDecision = true
		}
	}
	if !hasTurning {
		t.Error("ExpertValidate did not add turning point")
	}
	if !hasDecision {
		t.Error("ExpertValidate did not add decision point")
	}
}
