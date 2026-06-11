package geo_partitioner

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
)

type Partitioner struct {
	cfg *config.Config
	mu  sync.RWMutex

	lastRegions    []models.MilitaryRegion
	lastFCMResult  *models.FuzzyClusterResult
}

func New(cfg *config.Config) *Partitioner {
	return &Partitioner{cfg: cfg}
}

func (p *Partitioner) HaversineDist(lng1, lat1, lng2, lat2 float64) float64 {
	R := 6371.0
	rlat1 := lat1 * math.Pi / 180
	rlat2 := lat2 * math.Pi / 180
	dlat := rlat2 - rlat1
	dlng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(rlat1)*math.Cos(rlat2)*
			math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func (p *Partitioner) IDWInterpolate(dem []models.DEMTile, lng, lat float64) float64 {
	if len(dem) == 0 {
		return mockElevation(lng, lat)
	}
	var num, den float64
	power := 2.0
	for _, t := range dem {
		tlng := 73.0 + float64(t.TileX)*10 + 5
		tlat := 54.0 - float64(t.TileY)*10 - 5
		d := p.HaversineDist(lng, lat, tlng, tlat)
		if d < 1 {
			return float64(t.MinElev+t.MaxElev) / 2
		}
		w := 1.0 / math.Pow(d, power)
		avg := float64(t.MinElev+t.MaxElev) / 2
		num += w * avg
		den += w
	}
	if den < 1e-10 {
		return mockElevation(lng, lat)
	}
	return num / den
}

func (p *Partitioner) ComputeAccessibility(bf models.Battlefield, roads []models.AncientRoad) models.AccessibilityResult {
	minDist := math.Inf(1)
	totalScore := 0.0
	decay := p.cfg.Accessibility.DecayRate
	for _, r := range roads {
		for _, pt := range r.Coords {
			d := p.HaversineDist(bf.Lng, bf.Lat, pt[0], pt[1])
			if d < minDist {
				minDist = d
			}
			totalScore += math.Exp(-d * decay)
		}
	}
	score := 1.0 / (1.0 + minDist/10)
	level := "低"
	if score > 0.6 {
		level = "高"
	} else if score > 0.3 {
		level = "中"
	}
	return models.AccessibilityResult{
		NearestRoadDist:  minDist,
		AccessScore:      score,
		AccessLevel:      level,
		ConnectivityIndex: totalScore,
	}
}

func (p *Partitioner) GenerateTerrainProfile(
	dem []models.DEMTile,
	startLng, startLat, endLng, endLat float64,
	numPoints int,
) models.TerrainProfile {
	if numPoints <= 0 {
		numPoints = p.cfg.TerrainProfile.DefaultPoints
	}
	points := make([]models.ProfilePoint, numPoints)
	var minElev, maxElev, sumElev float64
	minElev = math.Inf(1)
	maxElev = math.Inf(-1)
	for i := 0; i < numPoints; i++ {
		t := float64(i) / float64(numPoints-1)
		lng := startLng + t*(endLng-startLng)
		lat := startLat + t*(endLat-startLat)
		elev := p.IDWInterpolate(dem, lng, lat)
		dist := p.HaversineDist(startLng, startLat, lng, lat)
		points[i] = models.ProfilePoint{
			Lng:       lng,
			Lat:       lat,
			Distance:  dist,
			Elevation: elev,
		}
		if elev < minElev {
			minElev = elev
		}
		if elev > maxElev {
			maxElev = elev
		}
		sumElev += elev
	}
	return models.TerrainProfile{
		StartLng:   startLng,
		StartLat:   startLat,
		EndLng:     endLng,
		EndLat:     endLat,
		NumPoints:  int32(numPoints),
		MinElev:    minElev,
		MaxElev:    maxElev,
		AvgElev:    sumElev / float64(numPoints),
		TotalDist:  p.HaversineDist(startLng, startLat, endLng, endLat),
		Points:     points,
	}
}

func (p *Partitioner) KMeansClustering(points [][]float64, k int) []int {
	if k <= 0 {
		k = p.cfg.Clustering.DefaultK
	}
	maxIter := p.cfg.Clustering.KM_MaxIter
	n := len(points)
	dim := len(points[0])
	labels := make([]int, n)
	centroids := make([][]float64, k)
	for i := 0; i < k; i++ {
		centroids[i] = make([]float64, dim)
		copy(centroids[i], points[i])
	}
	for iter := 0; iter < maxIter; iter++ {
		changed := false
		for i := 0; i < n; i++ {
			best := 0
			bestD := math.Inf(1)
			for c := 0; c < k; c++ {
				d := euclidean(points[i], centroids[c])
				if d < bestD {
					bestD = d
					best = c
				}
			}
			if labels[i] != best {
				labels[i] = best
				changed = true
			}
		}
		counts := make([]int, k)
		sums := make([][]float64, k)
		for i := 0; i < k; i++ {
			sums[i] = make([]float64, dim)
		}
		for i := 0; i < n; i++ {
			c := labels[i]
			counts[c]++
			for d := 0; d < dim; d++ {
				sums[c][d] += points[i][d]
			}
		}
		for c := 0; c < k; c++ {
			if counts[c] > 0 {
				for d := 0; d < dim; d++ {
					centroids[c][d] = sums[c][d] / float64(counts[c])
				}
			}
		}
		if !changed {
			break
		}
	}
	return labels
}

func (p *Partitioner) FuzzyCMeans(points [][]float64, k int) models.FuzzyClusterResult {
	if k <= 0 {
		k = p.cfg.Clustering.DefaultK
	}
	m := p.cfg.Clustering.FCM_Fuzzifier
	maxIter := p.cfg.Clustering.FCM_MaxIter
	eps := p.cfg.Clustering.FCM_Eps
	if m < 1.1 {
		m = 2.0
	}

	n := len(points)
	U := make([][]float64, n)
	for i := 0; i < n; i++ {
		U[i] = make([]float64, k)
		for j := 0; j < k; j++ {
			U[i][j] = randFloat()
		}
		normalizeRow(U[i])
	}
	dim := len(points[0])
	centroids := make([][]float64, k)
	for i := 0; i < k; i++ {
		centroids[i] = make([]float64, dim)
	}
	exp := 2.0 / (m - 1)

	for iter := 0; iter < maxIter; iter++ {
		for j := 0; j < k; j++ {
			num := make([]float64, dim)
			var den float64
			for i := 0; i < n; i++ {
				u := math.Pow(U[i][j], m)
				den += u
				for d := 0; d < dim; d++ {
					num[d] += u * points[i][d]
				}
			}
			if den > 1e-10 {
				for d := 0; d < dim; d++ {
					centroids[j][d] = num[d] / den
				}
			}
		}
		maxDiff := 0.0
		for i := 0; i < n; i++ {
			old := make([]float64, k)
			copy(old, U[i])
			distances := make([]float64, k)
			for j := 0; j < k; j++ {
				distances[j] = math.Max(0.0001, euclidean(points[i], centroids[j]))
			}
			for j := 0; j < k; j++ {
				var s float64
				for l := 0; l < k; l++ {
					s += math.Pow(distances[j]/distances[l], exp)
				}
				U[i][j] = 1.0 / s
			}
			for j := 0; j < k; j++ {
				if diff := math.Abs(U[i][j] - old[j]); diff > maxDiff {
					maxDiff = diff
				}
			}
		}
		if maxDiff < eps {
			break
		}
	}

	var pc, pe float64
	uncertainties := make([]float64, n)
	logK := math.Log(float64(k))
	for i := 0; i < n; i++ {
		var ent float64
		for j := 0; j < k; j++ {
			pc += U[i][j] * U[i][j]
			if U[i][j] > 1e-10 {
				ent += U[i][j] * math.Log(U[i][j])
			}
		}
		pe -= ent
		if logK > 0 {
			uncertainties[i] = (-ent) / logK
		}
	}
	pc /= float64(n)
	pe /= float64(n)

	hardLabels := make([]int, n)
	for i := 0; i < n; i++ {
		best := 0
		bestV := 0.0
		for j := 0; j < k; j++ {
			if U[i][j] > bestV {
				bestV = U[i][j]
				best = j
			}
		}
		hardLabels[i] = best
	}
	return models.FuzzyClusterResult{
		K:               k,
		Membership:      U,
		HardLabels:      hardLabels,
		Centroids:       centroids,
		PartitionCoef:   pc,
		PartitionEntropy: pe,
		Uncertainties:   uncertainties,
	}
}

func (p *Partitioner) GenerateMilitaryRegionsFCM(battlefields []models.Battlefield, k int) ([]models.MilitaryRegion, models.FuzzyClusterResult) {
	if k <= 0 {
		k = p.cfg.Clustering.DefaultK
	}
	scale := p.cfg.Clustering.TroopScale
	n := len(battlefields)
	points := make([][]float64, n)
	for i, bf := range battlefields {
		points[i] = []float64{bf.Lng, bf.Lat, float64(bf.TotalTroops) / scale}
	}
	fcm := p.FuzzyCMeans(points, k)
	regions := make([]models.MilitaryRegion, k)
	terrainOpts := []string{"山地", "平原", "河谷", "关隘"}
	terrainCount := 4
	for c := 0; c < k; c++ {
		var sumLng, sumLat, sumMem float64
		var battleCount int
		var densitySum float64
		terrainFreq := make([]int, terrainCount)
		for i := 0; i < n; i++ {
			if fcm.HardLabels[i] == c {
				mem := fcm.Membership[i][c]
				sumLng += battlefields[i].Lng * mem
				sumLat += battlefields[i].Lat * mem
				sumMem += mem
				battleCount++
				bfLng := battlefields[i].Lng
				bfLat := battlefields[i].Lat
				near := 0
				for j := 0; j < n; j++ {
					if j == i {
						continue
					}
					if p.HaversineDist(bfLng, bfLat, battlefields[j].Lng, battlefields[j].Lat) < 100 {
						near++
					}
				}
				densitySum += float64(near)
				switch battlefields[i].TerrainType {
				case "山地":
					terrainFreq[0]++
				case "平原":
					terrainFreq[1]++
				case "河谷":
					terrainFreq[2]++
				case "关隘":
					terrainFreq[3]++
				}
			}
		}
		var cx, cy float64
		avgMem := 0.0
		avgUnc := 0.0
		if sumMem > 0 {
			cx = sumLng / sumMem
			cy = sumLat / sumMem
			avgMem = sumMem / float64(battleCount)
			cnt := 0
			for i := 0; i < n; i++ {
				if fcm.HardLabels[i] == c {
					avgUnc += fcm.Uncertainties[i]
					cnt++
				}
			}
			if cnt > 0 {
				avgUnc /= float64(cnt)
			}
		}
		domIdx := 0
		for t := 1; t < terrainCount; t++ {
			if terrainFreq[t] > terrainFreq[domIdx] {
				domIdx = t
			}
		}
		radius := 1.5 + math.Log(1+float64(battleCount))*0.8
		coords := make([][2]float64, 20)
		for v := 0; v < 20; v++ {
			theta := float64(v) * 2 * math.Pi / 20
			coords[v] = [2]float64{
				cx + radius*math.Cos(theta)*1.1,
				cy + radius*math.Sin(theta),
			}
		}
		density := 0.0
		if battleCount > 0 {
			density = densitySum / float64(battleCount)
		}
		regions[c] = models.MilitaryRegion{
			ID:              int64(c + 1),
			RegionCode:      fmt.Sprintf("MR-%02d", c+1),
			RegionName:      fmt.Sprintf("军事分区_%d", c+1),
			BattleCount:     int32(battleCount),
			AvgDensity:      density,
			DominantTerrain: terrainOpts[domIdx],
			CenterLng:       cx,
			CenterLat:       cy,
			Coords:          [][][2]float64{coords},
			AvgMembership:   avgMem,
			Uncertainty:     avgUnc,
			PartitionCoef:   fcm.PartitionCoef,
			Entropy:         fcm.PartitionEntropy,
		}
	}
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].BattleCount > regions[j].BattleCount
	})
	for i := range regions {
		regions[i].RegionName = fmt.Sprintf("军事分区_%02d", i+1)
		regions[i].RegionCode = fmt.Sprintf("MR-%02d", i+1)
	}
	return regions, fcm
}

func (p *Partitioner) SetLast(regions []models.MilitaryRegion, fcm *models.FuzzyClusterResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastRegions = regions
	if fcm != nil {
		fr := *fcm
		p.lastFCMResult = &fr
	} else {
		p.lastFCMResult = nil
	}
}

func (p *Partitioner) GetLast() ([]models.MilitaryRegion, *models.FuzzyClusterResult) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastRegions, p.lastFCMResult
}

func euclidean(a, b []float64) float64 {
	var s float64
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return math.Sqrt(s)
}

func normalizeRow(row []float64) {
	var s float64
	for _, v := range row {
		s += v
	}
	if s > 0 {
		for i := range row {
			row[i] /= s
		}
	}
}

var rngState uint64 = 20240611

func randFloat() float64 {
	rngState = rngState*6364136223846793005 + 1442695040888963407
	return float64(rngState>>11) / (1 << 53)
}

func mockElevation(lng, lat float64) float64 {
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

var fmts = fmt.Sprintf

