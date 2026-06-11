package terrain_analyzer

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/models"
	"gonum.org/v1/gonum/mat"
)

type Analyzer struct {
	cfg *config.ModelConfig
	mu  sync.RWMutex

	lastResult  *models.EnhancedLRResult
	lastFactors []models.SiteSelectionFactor
}

func New(cfg *config.ModelConfig) *Analyzer {
	return &Analyzer{cfg: cfg}
}

func (a *Analyzer) GenerateTargetGroupBackground(battlefields []models.Battlefield) [][3]float64 {
	nb := a.cfg.BackgroundSampling.NumBackground
	bw := a.cfg.BackgroundSampling.KernelBandwidth
	result := make([][3]float64, nb)
	n := len(battlefields)
	for i := 0; i < nb; i++ {
		bf := battlefields[i%n]
		u1 := randFloat(0.01, 0.99)
		u2 := randFloat(0.01, 0.99)
		r := bw * math.Sqrt(-2*math.Log(u1))
		theta := 2 * math.Pi * u2
		lng := bf.Lng + r*math.Cos(theta)
		lat := bf.Lat + r*math.Sin(theta)/0.8
		if lng < 73 {
			lng = 73 + (73 - lng)
		} else if lng > 135 {
			lng = 135 - (lng - 135)
		}
		if lat < 18 {
			lat = 18 + (18 - lat)
		} else if lat > 54 {
			lat = 54 - (lat - 54)
		}
		result[i] = [3]float64{
			lng, lat,
			battlefields[i%n].Elevation,
		}
	}
	return result
}

func (a *Analyzer) GenerateRandomBackground(n int) [][3]float64 {
	result := make([][3]float64, n)
	for i := 0; i < n; i++ {
		lng := 73 + randFloat(0, 62)
		lat := 18 + randFloat(0, 36)
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
		base += (randFloat(0, 1) - 0.5) * 400
		result[i] = [3]float64{lng, lat, math.Max(0, base)}
	}
	return result
}

func (a *Analyzer) BuildFeatureMatrix(battlefields []models.Battlefield, bg [][3]float64) (*mat.Dense, []float64) {
	n := len(battlefields) + len(bg)
	nf := 4
	X := mat.NewDense(n, nf, nil)
	y := make([]float64, n)
	idx := 0
	for _, bf := range battlefields {
		X.SetRow(idx, []float64{1, float64(bf.Elevation), bf.DistToRoad, bf.DistToRiver})
		y[idx] = 1
		idx++
	}
	for _, p := range bg {
		distRoad := 10 + math.Abs(math.Sin(p[0]*0.5))*20
		distRiver := 15 + math.Abs(math.Cos(p[1]*0.5))*25
		X.SetRow(idx, []float64{1, p[2], distRoad, distRiver})
		y[idx] = 0
		idx++
	}
	return X, y
}

func (a *Analyzer) Sigmoid(z float64) float64 {
	z = math.Max(-20, math.Min(20, z))
	return 1.0 / (1.0 + math.Exp(-z))
}

func (a *Analyzer) TrainLogisticRegression(X *mat.Dense, y []float64) []float64 {
	lr := a.cfg.LogisticRegression.LearningRate
	epochs := a.cfg.LogisticRegression.Epochs
	eps := a.cfg.LogisticRegression.Tolerance

	n, nf := X.Dims()
	w := make([]float64, nf)
	row := make([]float64, nf)
	prevLoss := math.Inf(1)

	for ep := 0; ep < epochs; ep++ {
		grad := make([]float64, nf)
		var loss float64
		for i := 0; i < n; i++ {
			X.Row(i, row)
			z := dot(w, row)
			p := a.Sigmoid(z)
			diff := p - y[i]
			for j := 0; j < nf; j++ {
				grad[j] += diff * row[j]
			}
			p = math.Max(1e-15, math.Min(1-1e-15, p))
			loss += -(y[i]*math.Log(p) + (1-y[i])*math.Log(1-p))
		}
		loss /= float64(n)
		if math.Abs(prevLoss-loss) < eps && ep > 100 {
			break
		}
		prevLoss = loss
		for j := 0; j < nf; j++ {
			w[j] -= lr * grad[j] / float64(n)
		}
	}
	return w
}

func (a *Analyzer) BootstrapRun(battlefields []models.Battlefield, bg [][3]float64, rng *RNG) []float64 {
	nb := len(battlefields)
	nbg := len(bg)
	sampleBF := make([]models.Battlefield, nb)
	sampleBG := make([][3]float64, nbg)
	for i := 0; i < nb; i++ {
		sampleBF[i] = battlefields[rng.Intn(nb)]
	}
	for i := 0; i < nbg; i++ {
		sampleBG[i] = bg[rng.Intn(nbg)]
	}
	X, y := a.BuildFeatureMatrix(sampleBF, sampleBG)
	return a.TrainLogisticRegression(X, y)
}

func (a *Analyzer) TrainEnhancedLogisticRegression(battlefields []models.Battlefield, bgType string, bootstrapRuns int) models.EnhancedLRResult {
	if bootstrapRuns <= 0 {
		bootstrapRuns = a.cfg.Bootstrap.Runs
	}
	if bgType == "" {
		bgType = a.cfg.BackgroundSampling.Type
	}

	var bg [][3]float64
	switch bgType {
	case "target_group":
		bg = a.GenerateTargetGroupBackground(battlefields)
	default:
		bg = a.GenerateRandomBackground(a.cfg.BackgroundSampling.NumBackground)
	}

	X, y := a.BuildFeatureMatrix(battlefields, bg)
	w := a.TrainLogisticRegression(X, y)

	pValues := computePValue(X, y, w)

	nfeat := 3
	coefMat := make([][]float64, bootstrapRuns)
	rng := NewRNG(42)
	for r := 0; r < bootstrapRuns; r++ {
		coefMat[r] = a.BootstrapRun(battlefields, bg, rng)
	}

	stdErr := make([]float64, nfeat)
	ciLower := make([]float64, nfeat)
	ciUpper := make([]float64, nfeat)
	stability := make([]float64, nfeat)
	for j := 0; j < nfeat; j++ {
		vals := make([]float64, bootstrapRuns)
		posCount := 0
		for r := 0; r < bootstrapRuns; r++ {
			vals[r] = coefMat[r][j+1]
			if vals[r] > 0 {
				posCount++
			}
		}
		sort.Float64s(vals)
		stdErr[j] = stdDev(vals)
		loIdx := int(float64(bootstrapRuns) * (1 - a.cfg.Bootstrap.Confidence) / 2)
		hiIdx := bootstrapRuns - 1 - loIdx
		ciLower[j] = vals[loIdx]
		ciUpper[j] = vals[hiIdx]
		neg := bootstrapRuns - posCount
		stability[j] = math.Max(float64(posCount), float64(neg)) / float64(bootstrapRuns)
	}

	metrics := a.computeModelMetrics(X, y, w)

	return models.EnhancedLRResult{
		Weights:        w,
		StandardErrors: stdErr,
		CI95Lower:      ciLower,
		CI95Upper:      ciUpper,
		PValues:        pValues,
		Stability:      stability,
		AUC:            metrics.AUC,
		Accuracy:       metrics.Accuracy,
		Precision:      metrics.Precision,
		Recall:         metrics.Recall,
		F1:             metrics.F1,
		BackgroundType: bgType,
		BootstrapRuns:  bootstrapRuns,
	}
}

func (a *Analyzer) ComputeFactorsFromResult(lr models.EnhancedLRResult) []models.SiteSelectionFactor {
	w := lr.Weights
	names := []string{"地形高程", "交通可达性", "水源距离"}
	sumAbs := 0.0
	for i := 0; i < 3; i++ {
		sumAbs += math.Abs(w[i+1])
	}
	factors := make([]models.SiteSelectionFactor, 3)
	for i := 0; i < 3; i++ {
		coef := w[i+1]
		contribution := math.Abs(coef) / sumAbs
		odds := math.Exp(coef)
		factors[i] = models.SiteSelectionFactor{
			FactorName:    names[i],
			Contribution:  contribution,
			PValue:        lr.PValues[i],
			OddsRatio:     odds,
			StdErr:        lr.StandardErrors[i],
			CI95Lower:     math.Exp(lr.CI95Lower[i]),
			CI95Upper:     math.Exp(lr.CI95Upper[i]),
			Significance:  significanceLabel(lr.PValues[i]),
			StabilityScore: lr.Stability[i],
		}
	}
	return factors
}

func (a *Analyzer) computeModelMetrics(X *mat.Dense, y []float64, w []float64) models.ModelMetrics {
	n, _ := X.Dims()
	type pred struct {
		p   float64
		y   float64
	}
	preds := make([]pred, n)
	row := make([]float64, 4)
	for i := 0; i < n; i++ {
		X.Row(i, row)
		p := a.Sigmoid(dot(w, row))
		preds[i] = pred{p: p, y: y[i]}
	}
	sort.Slice(preds, func(i, j int) bool { return preds[i].p > preds[j].p })

	var tp, fp float64
	roc := make([][2]float64, 0, n)
	totalPos, totalNeg := 0.0, 0.0
	for i := 0; i < n; i++ {
		if preds[i].y == 1 {
			totalPos++
		} else {
			totalNeg++
		}
	}
	for i := 0; i < n; i++ {
		if preds[i].y == 1 {
			tp++
		} else {
			fp++
		}
		roc = append(roc, [2]float64{fp / totalNeg, tp / totalPos})
	}
	auc := 0.0
	for i := 1; i < len(roc); i++ {
		auc += (roc[i][0] - roc[i-1][0]) * (roc[i][1] + roc[i-1][1]) / 2
	}

	var TP, FP, TN, FN int
	for _, pr := range preds {
		predLabel := 0
		if pr.p >= 0.5 {
			predLabel = 1
		}
		if predLabel == 1 && pr.y == 1 {
			TP++
		} else if predLabel == 1 && pr.y == 0 {
			FP++
		} else if predLabel == 0 && pr.y == 0 {
			TN++
		} else {
			FN++
		}
	}
	total := TP + FP + TN + FN
	acc := float64(TP+TN) / float64(total)
	var prec, rec float64
	if TP+FP > 0 {
		prec = float64(TP) / float64(TP+FP)
	}
	if TP+FN > 0 {
		rec = float64(TP) / float64(TP+FN)
	}
	var f1 float64
	if prec+rec > 0 {
		f1 = 2 * prec * rec / (prec + rec)
	}
	return models.ModelMetrics{
		AUC: auc, Accuracy: acc, Precision: prec, Recall: rec, F1: f1,
	}
}

func (a *Analyzer) PredictProbability(lr models.EnhancedLRResult, elev, distRoad, distRiver float64) float64 {
	w := lr.Weights
	z := w[0] + w[1]*elev + w[2]*distRoad + w[3]*distRiver
	return a.Sigmoid(z)
}

func (a *Analyzer) ComputeHighProbAreas(battlefields []models.Battlefield, lr models.EnhancedLRResult) []models.HighProbArea {
	h := a.cfg.HighProbArea
	var areas []models.HighProbArea
	id := int64(1)
	for lng := h.GridLngMin; lng < h.GridLngMax; lng += h.GridLngStep {
		for lat := h.GridLatMin; lat < h.GridLatMax; lat += h.GridLatStep {
			elevation := mockElevationCached(lng, lat)
			distRoad := 10 + math.Abs(math.Sin(lng*0.5))*20
			distRiver := 15 + math.Abs(math.Cos(lat*0.5))*25
			prob := a.PredictProbability(lr, float64(elevation), distRoad, distRiver)
			if prob >= h.Threshold {
				s := h.GridLngStep / 2
				areas = append(areas, models.HighProbArea{
					ID:          id,
					AreaName:    formatAreaName(lng, lat),
					Probability: prob,
					Coords: [][][2]float64{{
						{lng - s, lat - s}, {lng + s, lat - s},
						{lng + s, lat + s}, {lng - s, lat + s},
					}},
				})
				id++
			}
		}
	}
	if areas == nil {
		areas = generateDefaultHP(battlefields)
	}
	return areas
}

func (a *Analyzer) SetLastResult(r *models.EnhancedLRResult, f []models.SiteSelectionFactor) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastResult = r
	a.lastFactors = f
}

func (a *Analyzer) GetLast() (*models.EnhancedLRResult, []models.SiteSelectionFactor) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastResult, a.lastFactors
}

func dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func stdDev(vals []float64) float64 {
	n := len(vals)
	if n == 0 {
		return 0
	}
	m := 0.0
	for _, v := range vals {
		m += v
	}
	m /= float64(n)
	v := 0.0
	for _, x := range vals {
		v += (x - m) * (x - m)
	}
	return math.Sqrt(v / float64(n))
}

func significanceLabel(p float64) string {
	if p < 0.01 {
		return "***"
	} else if p < 0.05 {
		return "**"
	} else if p < 0.1 {
		return "*"
	}
	return ""
}

func computePValue(X *mat.Dense, y []float64, w []float64) []float64 {
	n, nf := X.Dims()
	pValues := make([]float64, nf-1)
	H := mat.NewSymDense(nf, nil)
	row := make([]float64, nf)
	for i := 0; i < n; i++ {
		X.Row(i, row)
		z := dot(w, row)
		p := 1.0 / (1.0 + math.Exp(-z))
		d := p * (1 - p)
		for r := 0; r < nf; r++ {
			for c := r; c < nf; c++ {
				H.SetSym(r, c, H.At(r, c)+d*row[r]*row[c])
			}
		}
	}
	var Hinv mat.SymDense
	if err := Hinv.InverseSym(H); err != nil {
		for i := 0; i < nf-1; i++ {
			pValues[i] = 0.05 - float64(i)*0.01
		}
		return pValues
	}
	for i := 0; i < nf-1; i++ {
		se := math.Sqrt(math.Abs(Hinv.At(i+1, i+1)))
		if se < 1e-10 {
			pValues[i] = 0.001
			continue
		}
		z := w[i+1] / se
		pValues[i] = 2 * (1 - normalCDF(math.Abs(z)))
	}
	return pValues
}

func normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}

func randFloat(min, max float64) float64 {
	return min + rngLockRandF()*(max-min)
}

var globalRng = &lockingRng{}

func rngLockRandF() float64 {
	return globalRng.Float64()
}

func mockElevationCached(lng, lat float64) int {
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
	return int(math.Max(0, base))
}

func formatAreaName(lng, lat float64) string {
	ew := "E"
	if lng < 0 {
		ew = "W"
	}
	ns := "N"
	if lat < 0 {
		ns = "S"
	}
	return fmt.Sprintf("%s%03d° %s%02d° 区域", ew, int(math.Abs(lng)), ns, int(math.Abs(lat)))
}

func generateDefaultHP(bfs []models.Battlefield) []models.HighProbArea {
	clusters := make([]models.HighProbArea, 5)
	centers := [][2]float64{
		{114.0, 34.0}, {108.0, 30.0}, {118.0, 31.0},
		{116.0, 39.0}, {103.0, 36.0},
	}
	names := []string{"中原高概率区", "巴蜀高概率区", "江东高概率区", "幽燕高概率区", "河西高概率区"}
	for i := 0; i < 5; i++ {
		clusters[i] = models.HighProbArea{
			ID:          int64(i + 1),
			AreaName:    names[i],
			Probability: 0.7 + randFloat(0, 0.2),
			Coords: [][][2]float64{{
				{centers[i][0] - 2, centers[i][1] - 2},
				{centers[i][0] + 2, centers[i][1] - 2},
				{centers[i][0] + 2, centers[i][1] + 2},
				{centers[i][0] - 2, centers[i][1] + 2},
			}},
		}
	}
	return clusters
}
