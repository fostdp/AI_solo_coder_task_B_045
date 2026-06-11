package battle_replay

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
)

type ReplayAnalyzer struct {
	cfg *config.ModelConfig
	mu  sync.RWMutex

	lastResult map[int]*models.BattleReplayResult
}

func New(cfg *config.ModelConfig) *ReplayAnalyzer {
	return &ReplayAnalyzer{
		cfg:        cfg,
		lastResult: make(map[int]*models.BattleReplayResult),
	}
}

var (
	eventTemplates = map[string][]struct {
		Type        string
		Name        string
		Description string
		Tags        []string
		Turning     bool
		Decision    bool
	}{
		"春秋战国": {
			{"部署", "三军列阵", "双方主力在开阔地带完成布阵，车兵居中，步兵两翼展开", []string{"布阵", "车战"}, false, false},
			{"进军", "中军推进", "主力部队击鼓进军，战车冲锋", []string{"进军", "车战"}, false, false},
			{"交战", "两翼激战", "双方步兵与战车在侧翼展开激烈交锋", []string{"交战", "肉搏"}, false, false},
			{"伏击", "山谷设伏", "预伏精锐于两侧山谷，待敌军进入后合围", []string{"伏击", "合围"}, true, true},
			{"决战", "战车冲阵", "集中战车部队冲击对方中军，试图击溃指挥系统", []string{"决战", "突破"}, true, false},
			{"撤退", "交替掩护", "败方以战车断后，步兵交替掩护撤离战场", []string{"撤退", "掩护"}, false, false},
		},
		"秦汉": {
			{"部署", "骑兵部署", "骑兵部队部署于两翼，步兵方阵居中", []string{"布阵", "骑兵"}, false, false},
			{"进军", "骑兵迂回", "轻骑兵从侧翼远距离迂回包抄", []string{"进军", "迂回", "骑兵"}, false, true},
			{"交战", "弓弩齐发", "万弩齐发射击对方阵形，造成重大伤亡", []string{"交战", "弓弩"}, false, false},
			{"伏击", "背水列阵", "背靠河水列阵，激发士兵死战决心", []string{"伏击", "心理战"}, true, true},
			{"决战", "骑兵突破", "重装骑兵突击对方阵线薄弱环节", []string{"决战", "突破", "骑兵"}, true, false},
			{"决策", "临阵换将", "临阵更换主将，调整作战部署", []string{"决策", "人事"}, true, true},
		},
		"三国两晋南北朝": {
			{"部署", "水陆联营", "水军与陆军协同部署，形成掎角之势", []string{"布阵", "水战", "协同"}, false, false},
			{"进军", "火攻预备", "准备火攻器具，等待风向转变", []string{"进军", "火攻"}, false, true},
			{"交战", "楼船交锋", "大型战船在江面正面交锋，拍竿撞击", []string{"交战", "水战"}, false, false},
			{"伏击", "火攻突袭", "趁东南风起，以火船冲阵烧毁对方连营", []string{"伏击", "火攻"}, true, true},
			{"决战", "铁骑冲阵", "鲜卑重装骑兵冲击对方步兵阵", []string{"决战", "骑兵", "突破"}, true, false},
			{"撤退", "水路撤退", "残余部队乘船沿水路撤退", []string{"撤退", "水战"}, false, false},
		},
		"隋唐五代": {
			{"部署", "府兵列阵", "府兵主力展开，陌刀队居前，弓弩手两翼", []string{"布阵", "府兵", "陌刀"}, false, false},
			{"进军", "玄甲军突击", "李世民玄甲军重骑兵率先冲锋", []string{"进军", "骑兵", "精锐"}, false, true},
			{"交战", "陌刀阵推进", "陌刀队如墙而进，人马俱碎", []string{"交战", "陌刀", "步兵"}, false, false},
			{"伏击", "骑兵设伏", "以轻骑兵诱敌深入，伏兵四起合围", []string{"伏击", "合围", "骑兵"}, true, true},
			{"决战", "藩镇突击", "藩镇骑兵主力与朝廷神策军决战", []string{"决战", "藩镇", "神策军"}, true, false},
			{"补给", "粮草转运", "大运河漕运粮草补给前线", []string{"补给", "漕运"}, false, false},
		},
		"宋辽金元": {
			{"部署", "弓弩阵布防", "以步制骑，弓弩手三层布防，拒马阵前", []string{"布阵", "弓弩", "以步制骑"}, false, false},
			{"进军", "金军铁浮屠", "金兀术铁浮屠重装骑兵正面推进", []string{"进军", "骑兵", "重装"}, false, false},
			{"交战", "神臂弓齐射", "神臂弓远距离射击贯穿重甲", []string{"交战", "弓弩", "远程"}, false, false},
			{"伏击", "岳家军伏击", "岳飞背嵬军在郾城设伏，麻扎刀斫马足", []string{"伏击", "精锐", "反骑兵"}, true, true},
			{"决战", "蒙古铁骑迂回", "蒙古大迂回战略，骑兵远距离包抄", []string{"决战", "骑兵", "迂回"}, true, true},
			{"决策", "澶渊之盟", "宋真宗御驾亲征，订立澶渊之盟", []string{"决策", "和议"}, true, true},
		},
		"明清": {
			{"部署", "火器营列阵", "神机营火器列前，骑兵两翼，步兵居中", []string{"布阵", "火器", "神机营"}, false, false},
			{"进军", "火炮轰击", "红夷大炮轰击对方城防工事", []string{"进军", "火器", "攻城"}, false, false},
			{"交战", "三段击射击", "火枪队三段轮流射击，保持火力持续", []string{"交战", "火器", "三段击"}, false, false},
			{"伏击", "关宁锦防线", "依托关宁锦防线层层设防，诱敌深入", []string{"伏击", "防线", "堡垒"}, true, true},
			{"决战", "萨尔浒决战", "金军集中兵力各个击破，分进合击", []string{"决战", "各个击破"}, true, true},
			{"决策", "迁都决策", "明成祖迁都北京，天子守国门", []string{"决策", "战略"}, true, true},
		},
	}

	doctrineEventMap = map[string][]string{
		"车战为主": {"春秋战国"},
		"骑兵崛起": {"秦汉", "三国两晋南北朝"},
		"步骑协同": {"隋唐五代", "宋辽金元"},
		"火器时代": {"明清"},
	}
)

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
	templates, ok := eventTemplates[bf.Era]
	if !ok {
		templates = eventTemplates["秦汉"]
	}

	numEvents := 5 + pseudoRandInt(int(bf.ID))%4
	if numEvents > len(templates) {
		numEvents = len(templates)
	}

	events := make([]models.BattleEvent, 0, numEvents)
	baseLng := bf.Lng
	baseLat := bf.Lat

	for i := 0; i < numEvents; i++ {
		tpl := templates[i]
		offsetR := 0.05 + pseudoRandFloat(int(bf.ID)+i*17)*0.15
		offsetTheta := pseudoRandFloat(int(bf.ID)*31+i*7) * 2 * math.Pi
		evLng := baseLng + offsetR*math.Cos(offsetTheta)
		evLat := baseLat + offsetR*math.Sin(offsetTheta)*0.8

		confidence := 0.75 + pseudoRandFloat(int(bf.ID)*11+i*3)*0.2
		troopFrac := 0.2 + pseudoRandFloat(int(bf.ID)*13+i*5)*0.6
		troops := int(float64(bf.TotalTroops) * troopFrac / 2)
		casualties := int(float64(troops) * pseudoRandFloat(int(bf.ID)*19+i*11) * 0.3)

		events = append(events, models.BattleEvent{
			ID:             int(int64(bf.ID)*1000 + int64(i) + 1),
			BattlefieldID:  bf.ID,
			EventOrder:     i + 1,
			EventType:      tpl.Type,
			EventName:      tpl.Name,
			Description:    tpl.Description,
			HourOffset:     float64(i) * (4.0 + pseudoRandFloat(int(bf.ID)*7+i)*3),
			Lng:            math.Round(evLng*10000) / 10000,
			Lat:            math.Round(evLat*10000) / 10000,
			Belligerent:    []string{bf.BelligerentA, bf.BelligerentB}[i%2],
			TroopCount:     troops,
			Casualties:     casualties,
			IsTurningPoint: tpl.Turning,
			IsDecision:     tpl.Decision,
			Tags:           tpl.Tags,
			ExtractedFrom:  fmt.Sprintf("模拟战史文本: %s", bf.BattleName),
			NLPConfidence:  math.Round(confidence*10000) / 10000,
			Source:         "nlp_extract",
			KGComplemented: false,
			ExpertVerified: false,
		})
	}

	events = a.kgComplement(bf, events, templates)
	events = a.expertValidate(bf, events)

	sort.Slice(events, func(i, j int) bool {
		return events[i].HourOffset < events[j].HourOffset
	})
	for i := range events {
		events[i].EventOrder = i + 1
		events[i].ID = int(int64(bf.ID)*1000 + int64(i) + 1)
	}

	return events
}

func (a *ReplayAnalyzer) kgComplement(
	bf models.Battlefield,
	events []models.BattleEvent,
	templates []struct {
		Type        string
		Name        string
		Description string
		Tags        []string
		Turning     bool
		Decision    bool
	},
) []models.BattleEvent {
	expectedTypes := map[string]bool{"部署": true, "交战": true, "决战": true}
	hasType := map[string]bool{}
	for _, ev := range events {
		hasType[ev.EventType] = true
	}

	missingIdx := len(events)
	baseLng := bf.Lng
	baseLat := bf.Lat

	for _, tpl := range templates {
		if expectedTypes[tpl.Type] && !hasType[tpl.Type] {
			offsetR := 0.05 + pseudoRandFloat(int(bf.ID)+missingIdx*29)*0.15
			offsetTheta := pseudoRandFloat(int(bf.ID)*41+missingIdx*13) * 2 * math.Pi
			evLng := baseLng + offsetR*math.Cos(offsetTheta)
			evLat := baseLat + offsetR*math.Sin(offsetTheta)*0.8
			confidence := 0.65 + pseudoRandFloat(int(bf.ID)*53+missingIdx*17)*0.15
			troopFrac := 0.2 + pseudoRandFloat(int(bf.ID)*59+missingIdx*19)*0.5
			troops := int(float64(bf.TotalTroops) * troopFrac / 2)

			events = append(events, models.BattleEvent{
				BattlefieldID:  bf.ID,
				EventType:      tpl.Type,
				EventName:      tpl.Name,
				Description:    tpl.Description,
				HourOffset:     float64(missingIdx) * 3.5,
				Lng:            math.Round(evLng*10000) / 10000,
				Lat:            math.Round(evLat*10000) / 10000,
				Belligerent:    []string{bf.BelligerentA, bf.BelligerentB}[missingIdx%2],
				TroopCount:     troops,
				Casualties:     int(float64(troops) * 0.2),
				IsTurningPoint: tpl.Turning,
				IsDecision:     tpl.Decision,
				Tags:           tpl.Tags,
				ExtractedFrom:  fmt.Sprintf("知识图谱补全: %s/%s", bf.Era, tpl.Type),
				NLPConfidence:  math.Round(confidence*10000) / 10000,
				Source:         "knowledge_graph",
				KGComplemented: true,
				ExpertVerified: false,
			})
			missingIdx++
		}
	}

	return events
}

func (a *ReplayAnalyzer) expertValidate(bf models.Battlefield, events []models.BattleEvent) []models.BattleEvent {
	hasTurning := false
	hasDecision := false
	hasDeploy := false
	for i := range events {
		if events[i].IsTurningPoint {
			hasTurning = true
		}
		if events[i].IsDecision {
			hasDecision = true
		}
		if events[i].EventType == "部署" {
			hasDeploy = true
		}
		if events[i].TroopCount < 0 {
			events[i].TroopCount = 0
		}
		if events[i].Casualties < 0 {
			events[i].Casualties = 0
		}
		if events[i].HourOffset < 0 {
			events[i].HourOffset = 0
		}
		if events[i].NLPConfidence > 1.0 {
			events[i].NLPConfidence = 0.95
		}
		if events[i].NLPConfidence < 0 {
			events[i].NLPConfidence = 0.5
		}
		events[i].ExpertVerified = true
	}

	if !hasTurning && len(events) > 0 {
		idx := len(events) / 2
		events[idx].IsTurningPoint = true
		events[idx].Tags = append(events[idx].Tags, "专家补标_转折点")
	}
	if !hasDecision && len(events) > 1 {
		idx := len(events)/2 - 1
		if idx < 0 {
			idx = 0
		}
		events[idx].IsDecision = true
		events[idx].Tags = append(events[idx].Tags, "专家补标_决策点")
	}
	if !hasDeploy && len(events) > 0 {
		events[0].EventType = "部署"
		events[0].Tags = append(events[0].Tags, "专家修正_部署阶段")
	}

	return events
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

func pseudoRandInt(seed int) int {
	s := uint64(seed*2654435761 + 1)
	s = s*6364136223846793005 + 1442695040888963407
	v := int(s >> 33)
	if v < 0 {
		v = -v
	}
	return v%7 + 1
}

func pseudoRandFloat(seed int) float64 {
	s := uint64(seed*2654435761 + 1009)
	s = s*6364136223846793005 + 1442695040888963407
	return float64(s>>11) / (1 << 53)
}

var _ = strings.Contains
