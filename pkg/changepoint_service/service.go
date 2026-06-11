package changepoint_service

import (
	"math"
	"sort"

	"ancient-battlefield/pkg/models"
)

type ChangePointService struct{}

func New() *ChangePointService {
	return &ChangePointService{}
}

type YearValueWeight struct {
	Year   int
	Value  float64
	Weight float64
}

func (s *ChangePointService) detectChangePoints(series []YearValueWeight) []int {
	n := len(series)
	if n < 10 {
		return nil
	}

	points := make([]int, 0)
	window := 5

	for i := window; i < n-window; i++ {
		var preMean, postMean float64
		var preW, postW float64
		for k := i - window; k < i; k++ {
			preMean += series[k].Value * series[k].Weight
			preW += series[k].Weight
		}
		for k := i; k < i+window; k++ {
			postMean += series[k].Value * series[k].Weight
			postW += series[k].Weight
		}
		if preW > 0 {
			preMean /= preW
		}
		if postW > 0 {
			postMean /= postW
		}

		magnitude := math.Abs(postMean - preMean)
		threshold := 200.0
		if magnitude > threshold {
			if len(points) == 0 || i-points[len(points)-1] > 8 {
				points = append(points, i)
			}
		}
	}

	return points
}

func (s *ChangePointService) bayesianChangePoint(series []YearValueWeight) []int {
	n := len(series)
	if n < 6 {
		return nil
	}
	points := make([]int, 0)
	for i := 2; i < n-2; i++ {
		var preMean, postMean, preVar, postVar float64
		preN := float64(i)
		postN := float64(n - i)
		for k := 0; k < i; k++ {
			preMean += series[k].Value
		}
		for k := i; k < n; k++ {
			postMean += series[k].Value
		}
		preMean /= preN
		postMean /= postN
		for k := 0; k < i; k++ {
			preVar += (series[k].Value - preMean) * (series[k].Value - preMean)
		}
		for k := i; k < n; k++ {
			postVar += (series[k].Value - postMean) * (series[k].Value - postMean)
		}
		preVar /= math.Max(1, preN-1)
		postVar /= math.Max(1, postN-1)
		preVar += 100
		postVar += 100
		logBF := 0.5*math.Log(preVar/postVar) +
			(preVar+postVar+(postMean-preMean)*(postMean-preMean)*preN*postN/(preN+postN))/(2*(preVar+postVar))*0.5
		if math.Abs(logBF) > 0.3 {
			if len(points) == 0 || i-points[len(points)-1] > 5 {
				points = append(points, i)
			}
		}
	}
	return points
}

func (s *ChangePointService) cusumChangePoint(series []YearValueWeight) []int {
	n := len(series)
	if n < 8 {
		return nil
	}
	var mean float64
	for _, s := range series {
		mean += s.Value
	}
	mean /= float64(n)
	points := make([]int, 0)
	cusum := 0.0
	threshold := 800.0
	for i := 1; i < n; i++ {
		cusum = math.Max(0, cusum+(series[i].Value-mean-100))
		if cusum > threshold {
			if len(points) == 0 || i-points[len(points)-1] > 6 {
				points = append(points, i)
			}
			cusum = 0
		}
	}
	return points
}

func (s *ChangePointService) DetectAll(series []YearValueWeight) []int {
	windowCP := s.detectChangePoints(series)
	bayesCP := s.bayesianChangePoint(series)
	cusumCP := s.cusumChangePoint(series)

	methodHits := make(map[int]int)
	for _, p := range windowCP {
		methodHits[p]++
	}
	for _, p := range bayesCP {
		methodHits[p]++
	}
	for _, p := range cusumCP {
		methodHits[p]++
	}

	type hit struct {
		idx   int
		count int
	}
	var hits []hit
	for idx, count := range methodHits {
		hits = append(hits, hit{idx, count})
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].idx < hits[j].idx
	})

	result := make([]int, 0)
	for _, h := range hits {
		result = append(result, h.idx)
	}
	return result
}

func pseudoRandFloat(seed int) float64 {
	s := uint64(seed*2654435761 + 1009)
	s = s*6364136223846793005 + 1442695040888963407
	return float64(s>>11) / (1 << 53)
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

var _ = models.ChangePoint{}
