package battle_replayer

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
	"ancient-battlefield/pkg/nlp_worker"
)

type ReplayAnalyzer struct {
	cfg        *config.ModelConfig
	mu         sync.RWMutex
	nlpWorker  *nlp_worker.NLPWorker
	lastResult map[int]*models.BattleReplayResult
}

func New(cfg *config.ModelConfig) *ReplayAnalyzer {
	return &ReplayAnalyzer{
		cfg:        cfg,
		nlpWorker:  nlp_worker.New(cfg),
		lastResult: make(map[int]*models.BattleReplayResult),
	}
}

func (a *ReplayAnalyzer) mockElev(lng, lat float64) float64 {
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
	return math.Max(0, base)
}

func (a *ReplayAnalyzer) NLPExtractEvents(bf models.Battlefield) []models.BattleEvent {
	return a.nlpWorker.ExtractEvents(bf)
}

func (a *ReplayAnalyzer) BuildTimeline(bf models.Battlefield, events []models.BattleEvent) models.CampaignTimeline {
	var totalDuration float64
	if len(events) > 0 {
		totalDuration = events[len(events)-1].HourOffset + 2.0
	}

	turningPoints := make([]models.BattleEvent, 0)
	decisions := make([]models.BattleEvent, 0)
	for _, ev := range events {
		if ev.IsTurningPoint {
			turningPoints = append(turningPoints, ev)
		}
		if ev.IsDecision {
			decisions = append(decisions, ev)
		}
	}

	return models.CampaignTimeline{
		BattlefieldID:  bf.ID,
		BattleName:     bf.BattleName,
		TotalDurationH: math.Round(totalDuration*100) / 100,
		Events:         events,
		TurningPoints:  turningPoints,
		Decisions:      decisions,
	}
}

func (a *ReplayAnalyzer) GenerateAnimationFrames(
	bf models.Battlefield,
	timeline models.CampaignTimeline,
	fps int,
) []models.AnimationFrame {
	if fps <= 0 {
		fps = 12
	}

	duration := timeline.TotalDurationH
	if duration <= 0 {
		duration = 24
	}
	totalFrames := int(math.Max(20, float64(fps)*8))

	frames := make([]models.AnimationFrame, 0, totalFrames)
	baseLng := bf.Lng
	baseLat := bf.Lat

	sideA := bf.BelligerentA
	sideB := bf.BelligerentB

	unitsA := []string{"左军", "中军", "右军", "前军", "后军"}
	unitsB := []string{"左翼", "主力", "右翼", "前锋", "后卫"}

	for f := 0; f < totalFrames; f++ {
		t := float64(f) / float64(totalFrames-1)
		ts := t * duration
		hoursInt := int(ts)
		minsInt := int((ts - float64(hoursInt)) * 60)
		timeLabel := fmt.Sprintf("T+%02d:%02d", hoursInt, minsInt)

		numUnitsA := 3
		numUnitsB := 3
		if t > 0.3 {
			numUnitsA = 4
			numUnitsB = 4
		}
		if t > 0.6 {
			numUnitsA = 5
			numUnitsB = 5
		}

		positions := make([]struct {
			Belligerent string  `json:"belligerent"`
			UnitName    string  `json:"unit_name"`
			Lng         float64 `json:"lng"`
			Lat         float64 `json:"lat"`
			TroopCount  int     `json:"troop_count"`
			IconType    string  `json:"icon_type"`
		}, 0, numUnitsA+numUnitsB)

		for i := 0; i < numUnitsA; i++ {
			phase := t * math.Pi
			advanceA := t * 0.08
			if t > 0.7 {
				advanceA = 0.08 - (t-0.7)*0.05
			}
			lngA := baseLng - 0.15 + advanceA + math.Sin(phase+float64(i))*0.03
			latA := baseLat + (float64(i)-float64(numUnitsA-1)/2.0)*0.06
			troopA := int(float64(bf.TroopA) / float64(numUnitsA) * (1 - t*0.4))
			positions = append(positions, struct {
				Belligerent string  `json:"belligerent"`
				UnitName    string  `json:"unit_name"`
				Lng         float64 `json:"lng"`
				Lat         float64 `json:"lat"`
				TroopCount  int     `json:"troop_count"`
				IconType    string  `json:"icon_type"`
			}{sideA, unitsA[i], math.Round(lngA*10000) / 10000, math.Round(latA*10000) / 10000, troopA, "infantry"})
		}

		for i := 0; i < numUnitsB; i++ {
			phase := t*math.Pi + 0.5
			advanceB := t * 0.06
			if t > 0.6 {
				advanceB = 0.06 + (t-0.6)*0.08
			}
			lngB := baseLng + 0.15 - advanceB + math.Sin(phase+float64(i))*0.03
			latB := baseLat + (float64(i)-float64(numUnitsB-1)/2.0)*0.06
			troopB := int(float64(bf.TroopB) / float64(numUnitsB) * (1 - t*0.25))
			iconB := "infantry"
			if t > 0.25 && i%2 == 0 {
				iconB = "cavalry"
			}
			positions = append(positions, struct {
				Belligerent string  `json:"belligerent"`
				UnitName    string  `json:"unit_name"`
				Lng         float64 `json:"lng"`
				Lat         float64 `json:"lat"`
				TroopCount  int     `json:"troop_count"`
				IconType    string  `json:"icon_type"`
			}{sideB, unitsB[i], math.Round(lngB*10000) / 10000, math.Round(latB*10000) / 10000, troopB, iconB})
		}

		var activeEvent *models.BattleEvent
		for _, ev := range timeline.Events {
			if ts >= ev.HourOffset && ts < ev.HourOffset+1.5 {
				evCopy := ev
				activeEvent = &evCopy
				break
			}
		}

		var frontLines [][][2]float64
		if t > 0.2 && t < 0.85 {
			frontLine := make([][2]float64, 8)
			for i := 0; i < 8; i++ {
				lngF := baseLng + t*0.02 + math.Sin(float64(i)*0.8+t*math.Pi)*0.015
				latF := baseLat - 0.2 + float64(i)*0.057
				frontLine[i] = [2]float64{lngF, latF}
			}
			frontLines = [][][2]float64{frontLine}
		}

		frames = append(frames, models.AnimationFrame{
			FrameIndex:     f,
			TimestampH:     math.Round(ts*100) / 100,
			TimeLabel:      timeLabel,
			TroopPositions: positions,
			ActiveEvent:    activeEvent,
			FrontLines:     frontLines,
		})
	}

	return frames
}

func (a *ReplayAnalyzer) GenerateBattleReplay(bf models.Battlefield, fps int) models.BattleReplayResult {
	if fps <= 0 {
		fps = 12
	}

	events := a.NLPExtractEvents(bf)
	timeline := a.BuildTimeline(bf, events)
	frames := a.GenerateAnimationFrames(bf, timeline, fps)

	avgConf := 0.0
	kgCount := 0
	expertPass := 0
	for _, ev := range events {
		avgConf += ev.NLPConfidence
		if ev.KGComplemented {
			kgCount++
		}
		if ev.ExpertVerified {
			expertPass++
		}
	}
	if len(events) > 0 {
		avgConf /= float64(len(events))
	}
	avgConf = math.Round(avgConf*10000) / 10000
	kgRate := 0.0
	expertRate := 0.0
	if len(events) > 0 {
		kgRate = math.Round(float64(kgCount)/float64(len(events))*10000) / 10000
		expertRate = math.Round(float64(expertPass)/float64(len(events))*10000) / 10000
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

	result := models.BattleReplayResult{
		Timeline:    timeline,
		Frames:      frames,
		Fps:         fps,
		TotalFrames: len(frames),
	}
	result.NLPStats.TotalEvents = len(events)
	result.NLPStats.AvgConfidence = avgConf
	result.NLPStats.TurningPointCount = tpCount
	result.NLPStats.DecisionCount = dcCount
	result.KGStats.ComplementedCount = kgCount
	result.KGStats.ComplementedRate = kgRate
	result.KGStats.ExpertPassRate = expertRate

	a.mu.Lock()
	a.lastResult[bf.ID] = &result
	a.mu.Unlock()

	return result
}

func (a *ReplayAnalyzer) GetLast(bfID int) *models.BattleReplayResult {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastResult[bfID]
}

var _ = strings.Contains
