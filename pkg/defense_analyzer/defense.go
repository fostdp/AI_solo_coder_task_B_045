package defense_analyzer

import (
	"math"
	"sort"
	"sync"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
)

type DefenseAnalyzer struct {
	cfg *config.ModelConfig
	mu  sync.RWMutex

	lastResult map[int]*models.DefenseEvaluation
}

func New(cfg *config.ModelConfig) *DefenseAnalyzer {
	return &DefenseAnalyzer{
		cfg:        cfg,
		lastResult: make(map[int]*models.DefenseEvaluation),
	}
}

func (a *DefenseAnalyzer) haversineKm(lng1, lat1, lng2, lat2 float64) float64 {
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

func mockElev(lng, lat float64) float64 {
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
	return math.Max(0, base+math.Sin(lng*2)*math.Cos(lat*3)*80)
}

var structureTemplates = map[string][]struct {
	Type      string
	Name      string
	Height    float64
	Length    float64
	Thickness float64
	Material  string
	Gates     int
	Towers    int
}{
	"城墙": {
		{"城墙", "主城城墙", 12, 3500, 6, "夯土包砖", 4, 24},
		{"城墙", "外郭城墙", 8, 8000, 4, "夯土", 6, 36},
	},
	"关隘": {
		{"关隘", "主关隘", 10, 500, 5, "条石", 2, 8},
	},
	"堡垒": {
		{"堡垒", "烽火堡垒", 8, 80, 3, "夯土", 1, 4},
	},
	"烽火台": {
		{"烽火台", "瞭望烽火台", 15, 20, 3, "砖石", 0, 1},
	},
	"要塞": {
		{"要塞", "边防要塞", 14, 1200, 7, "条石包砖", 3, 16},
	},
	"寨堡": {
		{"寨堡", "驻军寨堡", 7, 600, 4, "夯土木栅", 2, 6},
	},
	"护城河": {
		{"护城河", "城壕", 4, 3000, 15, "水壕", 0, 0},
	},
}

func (a *DefenseAnalyzer) GenerateMilitaryStructures(bf models.Battlefield) []models.MilitaryStructure {
	seed := bf.ID
	result := make([]models.MilitaryStructure, 0)

	count := 3 + pseudoRandInt(seed)%3
	structureKeys := []string{"城墙", "关隘", "堡垒", "烽火台", "要塞", "寨堡", "护城河"}

	for i := 0; i < count; i++ {
		keyIdx := (i + pseudoRandInt(seed+i*7)) % len(structureKeys)
		key := structureKeys[keyIdx]
		templates := structureTemplates[key]
		if len(templates) == 0 {
			continue
		}
		tpl := templates[pseudoRandInt(seed+i*11)%len(templates)]

		offsetR := 0.02 + pseudoRandFloat(seed+i*5)*0.1
		offsetTheta := pseudoRandFloat(seed+i*13) * 2 * math.Pi
		lng := bf.Lng + offsetR*math.Cos(offsetTheta)
		lat := bf.Lat + offsetR*math.Sin(offsetTheta)*0.8

		r := 0.015 + pseudoRandFloat(seed+i*17)*0.025
		n := 16
		polyCoords := make([][2]float64, n+1)
		for k := 0; k < n; k++ {
			theta := float64(k) / float64(n) * 2 * math.Pi
			rr := r * (0.7 + pseudoRandFloat(seed*3+i*3+k)*0.6)
			polyCoords[k] = [2]float64{
				math.Round((lng+rr*math.Cos(theta))*10000) / 10000,
				math.Round((lat+rr*math.Sin(theta))*10000) / 10000,
			}
		}
		polyCoords[n] = polyCoords[0]

		coords := [][][2]float64{polyCoords}

		heightVar := 0.8 + pseudoRandFloat(seed+i*19)*0.4
		lengthVar := 0.7 + pseudoRandFloat(seed+i*23)*0.6

		result = append(result, models.MilitaryStructure{
			ID:             seed*100 + i + 1,
			StructureName:  bf.BattleName + "-" + tpl.Name,
			StructureType:  tpl.Type,
			BattlefieldID:  bf.ID,
			Dynasty:        bf.Dynasty,
			Lng:            math.Round(lng*10000) / 10000,
			Lat:            math.Round(lat*10000) / 10000,
			HeightM:        math.Round(tpl.Height*heightVar*100) / 100,
			LengthM:        math.Round(tpl.Length*lengthVar*100) / 100,
			ThicknessM:     math.Round(tpl.Thickness*100) / 100,
			Material:       tpl.Material,
			GateCount:      tpl.Gates,
			TowerCount:     tpl.Towers,
			Coords:         coords,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].HeightM > result[j].HeightM
	})

	return result
}

type viewshedSample struct {
	dirDeg      float64
	dirName     string
	distanceKm  float64
	visiblePct  float64
	obstructed  bool
}

func (a *DefenseAnalyzer) lineOfSight(
	srcLng, srcLat float64,
	srcHeightM float64,
	dirDeg float64,
	maxDistKm float64,
) viewshedSample {
	toRad := math.Pi / 180.0
	dirRad := dirDeg * toRad

	steps := 20
	visibleCount := 0
	maxSeen := 0.0
	srcElev := mockElev(srcLng, srcLat) + srcHeightM

	for s := 1; s <= steps; s++ {
		frac := float64(s) / float64(steps)
		dist := maxDistKm * frac
		deltaLat := dist / 111.0 * math.Cos(dirRad)
		deltaLng := dist / 111.0 * math.Sin(dirRad) / math.Cos(srcLat*toRad)
		targetLng := srcLng + deltaLng
		targetLat := srcLat + deltaLat
		targetElev := mockElev(targetLng, targetLat)

		slope := (targetElev - srcElev) / (dist * 1000)
		obstructionSlope := 0.0005

		if slope <= obstructionSlope {
			visibleCount++
			maxSeen = math.Max(maxSeen, frac)
		} else {
			break
		}
	}

	dirNames := []string{"北", "东北", "东", "东南", "南", "西南", "西", "西北"}
	idx := int(math.Round(dirDeg/45.0)) % 8
	if idx < 0 {
		idx += 8
	}

	return viewshedSample{
		dirDeg:     dirDeg,
		dirName:    dirNames[idx],
		distanceKm: maxSeen * maxDistKm,
		visiblePct: float64(visibleCount) / float64(steps),
		obstructed: visibleCount < steps,
	}
}

func (a *DefenseAnalyzer) computeViewshed(
	structure models.MilitaryStructure,
	sampleKm float64,
) ([]viewshedSample, float64) {
	maxDist := 5.0 + structure.HeightM/2.0
	if sampleKm > 0 {
		maxDist = sampleKm
	}

	numDirs := 36
	samples := make([]viewshedSample, numDirs)
	totalVisible := 0.0

	for i := 0; i < numDirs; i++ {
		deg := float64(i) * (360.0 / float64(numDirs))
		samples[i] = a.lineOfSight(
			structure.Lng, structure.Lat,
			structure.HeightM,
			deg, maxDist,
		)
		totalVisible += samples[i].visiblePct
	}

	avgVis := totalVisible / float64(numDirs)
	return samples, math.Round(avgVis*10000) / 10000
}

func (a *DefenseAnalyzer) identifyBlindZones(
	structure models.MilitaryStructure,
	samples []viewshedSample,
) []models.DefenseBlindZone {
	blindZones := make([]models.DefenseBlindZone, 0)
	numDirs := len(samples)
	if numDirs == 0 {
		return blindZones
	}

	i := 0
	for i < numDirs {
		s := samples[i]
		if s.visiblePct < 0.5 {
			startIdx := i
			endIdx := i
			for endIdx+1 < numDirs && samples[endIdx+1].visiblePct < 0.5 &&
				(endIdx+1-startIdx) < numDirs/4 {
				endIdx++
			}

			midIdx := (startIdx + endIdx) / 2
			centerDeg := samples[midIdx].dirDeg
			toRad := math.Pi / 180.0
			dirRad := centerDeg * toRad
			avgDist := 0.0
			minVis := 1.0
			for k := startIdx; k <= endIdx; k++ {
				avgDist += samples[k].distanceKm
				if samples[k].visiblePct < minVis {
					minVis = samples[k].visiblePct
				}
			}
			spanWidth := endIdx - startIdx + 1
			avgDist /= float64(spanWidth)

			radiusDeg := avgDist / 111.0
			centerLat := structure.Lat + radiusDeg*math.Cos(dirRad)*0.5
			centerLng := structure.Lng + radiusDeg*math.Sin(dirRad)*0.5/math.Cos(structure.Lat*toRad)

			spanRad := float64(spanWidth) * (360.0 / float64(numDirs)) * toRad
			n := 12
			polyCoords := make([][2]float64, n+1)
			for k := 0; k < n; k++ {
				theta := -spanRad/2 + float64(k)/float64(n-1)*spanRad
				rr := radiusDeg * (0.5 + pseudoRandFloat(structure.ID*3+k)*0.5)
				polyCoords[k] = [2]float64{
					math.Round((centerLng+rr*math.Sin(dirRad+theta))*10000) / 10000,
					math.Round((centerLat+rr*math.Cos(dirRad+theta))*10000) / 10000,
				}
			}
			polyCoords[n] = polyCoords[0]

			areaKm2 := math.Pi * avgDist * avgDist * float64(spanWidth) / float64(numDirs)
			riskLevel := "低"
			switch {
			case minVis < 0.15:
				riskLevel = "极高"
			case minVis < 0.25:
				riskLevel = "高"
			case minVis < 0.4:
				riskLevel = "中"
			}

			blindZones = append(blindZones, models.DefenseBlindZone{
				ID:             structure.ID*100 + len(blindZones) + 1,
				StructureID:    structure.ID,
				CenterLng:      math.Round(centerLng*10000) / 10000,
				CenterLat:      math.Round(centerLat*10000) / 10000,
				AreaKm2:        math.Round(areaKm2*10000) / 10000,
				Direction:      samples[midIdx].dirName,
				MaxDistanceKm:  math.Round(avgDist*100) / 100,
				VisibilityPct:  math.Round(minVis*10000) / 10000,
				RiskLevel:      riskLevel,
				Coords:         [][][2]float64{polyCoords},
			})

			i = endIdx + 1
		} else {
			i++
		}
	}

	sort.Slice(blindZones, func(i, j int) bool {
		return blindZones[i].AreaKm2 > blindZones[j].AreaKm2
	})

	return blindZones
}

func (a *DefenseAnalyzer) structuralScore(s models.MilitaryStructure) float64 {
	hScore := math.Min(1, s.HeightM/15.0)
	lScore := math.Min(1, s.LengthM/5000.0)
	tScore := math.Min(1, s.ThicknessM/8.0)
	materialBonus := map[string]float64{
		"条石包砖": 1.0, "条石": 0.9, "夯土包砖": 0.8,
		"砖石": 0.75, "夯土": 0.6, "夯土木栅": 0.4, "水壕": 0.7,
	}
	mScore := materialBonus[s.Material]
	gScore := math.Min(1, float64(s.TowerCount)/20.0)

	return math.Round((hScore*0.3+lScore*0.2+tScore*0.2+mScore*0.2+gScore*0.1)*10000) / 10000
}

func (a *DefenseAnalyzer) topographicScore(s models.MilitaryStructure, bf models.Battlefield) float64 {
	elev := mockElev(s.Lng, s.Lat)
	baseElev := mockElev(bf.Lng, bf.Lat)
	elevDiff := elev - baseElev
	elevScore := math.Max(0, math.Min(1, 0.5+elevDiff/200.0))
	riverBonus := 1.0
	if bf.DistanceToRiver < 3 {
		riverBonus = 1.15
	}
	roadBonus := 1.0
	if bf.DistanceToRoad < 2 {
		roadBonus = 1.1
	}
	score := elevScore * riverBonus * roadBonus
	return math.Round(math.Min(1, score)*10000) / 10000
}

func (a *DefenseAnalyzer) makeRecommendations(
	blindZones []models.DefenseBlindZone,
	structural float64,
	topographic float64,
	avgVis float64,
) []string {
	recs := make([]string, 0)
	if avgVis < 0.7 {
		recs = append(recs, "整体视野不足，建议增设瞭望塔")
	}
	for _, bz := range blindZones {
		if bz.RiskLevel == "极高" || bz.RiskLevel == "高" {
			recs = append(recs, bz.Direction+"方向存在"+bz.RiskLevel+"防御盲区，建议增设烽火台或前哨")
		}
		if len(recs) >= 3 {
			break
		}
	}
	if structural < 0.5 {
		recs = append(recs, "结构强度较弱，建议加固城墙厚度与高度")
	}
	if topographic < 0.4 {
		recs = append(recs, "地形优势不足，建议利用周边地形改造防御工事")
	}
	if len(recs) == 0 {
		recs = append(recs, "防御体系整体良好，建议定期维护")
	}
	return recs
}

func (a *DefenseAnalyzer) EvaluateStructure(
	structure models.MilitaryStructure,
	bf models.Battlefield,
	viewshedSampleKm float64,
) models.DefenseEvaluation {
	if viewshedSampleKm <= 0 {
		viewshedSampleKm = 8.0
	}

	samples, avgVis := a.computeViewshed(structure, viewshedSampleKm)
	blindZones := a.identifyBlindZones(structure, samples)

	totalArea := 0.0
	for _, bz := range blindZones {
		totalArea += bz.AreaKm2
	}

	structural := a.structuralScore(structure)
	topographic := a.topographicScore(structure, bf)
	visibility := avgVis

	overall := math.Round((visibility*0.4+structural*0.3+topographic*0.3)*10000) / 10000
	recs := a.makeRecommendations(blindZones, structural, topographic, avgVis)

	result := models.DefenseEvaluation{
		StructureID:       structure.ID,
		StructureName:     structure.StructureName,
		OverallScore:      overall,
		VisibilityScore:   visibility,
		StructuralScore:   structural,
		TopographicScore:  topographic,
		BlindZoneCount:    len(blindZones),
		TotalBlindAreaKm2: math.Round(totalArea*10000) / 10000,
		AvgVisibilityPct:  avgVis,
		BlindZones:        blindZones,
		ViewshedSampleKm:  viewshedSampleKm,
		Recommendations:   recs,
	}

	a.mu.Lock()
	a.lastResult[structure.ID] = &result
	a.mu.Unlock()

	return result
}

func (a *DefenseAnalyzer) EvaluateAll(
	bf models.Battlefield,
	viewshedSampleKm float64,
) []models.DefenseEvaluation {
	structures := a.GenerateMilitaryStructures(bf)
	results := make([]models.DefenseEvaluation, 0, len(structures))
	for _, s := range structures {
		results = append(results, a.EvaluateStructure(s, bf, viewshedSampleKm))
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].OverallScore > results[j].OverallScore
	})
	return results
}

func (a *DefenseAnalyzer) GetLast(structureID int) *models.DefenseEvaluation {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastResult[structureID]
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
