package changepoint_service

import (
	"testing"
	"time"
)

func makeSeries(values []float64) []YearValueWeight {
	s := make([]YearValueWeight, len(values))
	for i, v := range values {
		s[i] = YearValueWeight{Year: -700 + i*50, Value: v, Weight: 1.0}
	}
	return s
}

func TestNew_ReturnsNonNull(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New returned nil")
	}
}

func TestDetectChangePoints_SufficientData(t *testing.T) {
	s := New()
	values := make([]float64, 25)
	for i := range values {
		if i < 12 {
			values[i] = 100
		} else {
			values[i] = 800
		}
	}
	series := makeSeries(values)
	result := s.detectChangePoints(series)
	if result == nil {
		t.Error("expected non-nil result for sufficient data")
	}
}

func TestDetectChangePoints_InsufficientData(t *testing.T) {
	s := New()
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}
	series := makeSeries(values)
	result := s.detectChangePoints(series)
	if result != nil && len(result) > 0 {
		t.Errorf("expected nil or empty for <10 elements, got %v", result)
	}
}

func TestDetectChangePoints_NoChange(t *testing.T) {
	s := New()
	values := make([]float64, 20)
	for i := range values {
		values[i] = 500
	}
	series := makeSeries(values)
	result := s.detectChangePoints(series)
	if len(result) > 0 {
		t.Errorf("expected no change points for constant series, got %v", result)
	}
}

func TestDetectChangePoints_SingleJump(t *testing.T) {
	s := New()
	values := make([]float64, 30)
	for i := range values {
		if i < 15 {
			values[i] = 100
		} else {
			values[i] = 800
		}
	}
	series := makeSeries(values)
	result := s.detectChangePoints(series)
	if len(result) == 0 {
		t.Error("expected at least one change point for jump series")
	}
}

func TestBayesianChangePoint_SmallSample(t *testing.T) {
	s := New()
	values := []float64{10, 20, 30, 40, 50, 800, 900}
	series := makeSeries(values)
	result := s.bayesianChangePoint(series)
	_ = result
}

func TestBayesianChangePoint_TooSmall(t *testing.T) {
	s := New()
	values := []float64{1, 2, 3, 4, 5}
	series := makeSeries(values)
	result := s.bayesianChangePoint(series)
	if result != nil {
		t.Errorf("expected nil for <6 elements, got %v", result)
	}
}

func TestCusumChangePoint_SufficientData(t *testing.T) {
	s := New()
	values := make([]float64, 15)
	for i := range values {
		if i < 7 {
			values[i] = 100
		} else {
			values[i] = 1500
		}
	}
	series := makeSeries(values)
	result := s.cusumChangePoint(series)
	_ = result
}

func TestCusumChangePoint_TooSmall(t *testing.T) {
	s := New()
	values := []float64{1, 2, 3, 4, 5, 6, 7}
	series := makeSeries(values)
	result := s.cusumChangePoint(series)
	if result != nil {
		t.Errorf("expected nil for <8 elements, got %v", result)
	}
}

func TestDetectAll_SufficientData(t *testing.T) {
	s := New()
	values := make([]float64, 30)
	for i := range values {
		if i < 15 {
			values[i] = 100
		} else {
			values[i] = 800
		}
	}
	series := makeSeries(values)
	result := s.DetectAll(series)
	_ = result
}

func TestDetectAll_SmallSample(t *testing.T) {
	s := New()
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120}
	series := makeSeries(values)
	result := s.DetectAll(series)
	_ = result
}

func TestDetectAll_NoChange(t *testing.T) {
	s := New()
	values := make([]float64, 25)
	for i := range values {
		values[i] = 500
	}
	series := makeSeries(values)
	result := s.DetectAll(series)
	if len(result) > 2 {
		t.Errorf("expected few or no change points for constant series, got %d", len(result))
	}
}

func TestDetectAll_Determinism(t *testing.T) {
	s := New()
	values := make([]float64, 25)
	for i := range values {
		if i < 12 {
			values[i] = 100
		} else {
			values[i] = 800
		}
	}
	series := makeSeries(values)
	result1 := s.DetectAll(series)
	result2 := s.DetectAll(series)
	if len(result1) != len(result2) {
		t.Fatalf("determinism: len %d vs %d", len(result1), len(result2))
	}
	for i := range result1 {
		if result1[i] != result2[i] {
			t.Errorf("result[%d] mismatch: %d vs %d", i, result1[i], result2[i])
		}
	}
}

func TestYearValueWeight_Type(t *testing.T) {
	yvw := YearValueWeight{Year: -500, Value: 100.0, Weight: 1.0}
	if yvw.Year != -500 {
		t.Errorf("Year = %d, want -500", yvw.Year)
	}
	if yvw.Value != 100.0 {
		t.Errorf("Value = %f, want 100.0", yvw.Value)
	}
	if yvw.Weight != 1.0 {
		t.Errorf("Weight = %f, want 1.0", yvw.Weight)
	}
}

func TestDetectAll_LargeSeries(t *testing.T) {
	s := New()
	values := make([]float64, 120)
	for i := range values {
		if i < 40 {
			values[i] = 100
		} else if i < 80 {
			values[i] = 800
		} else {
			values[i] = 300
		}
	}
	series := makeSeries(values)
	start := time.Now()
	result := s.DetectAll(series)
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Errorf("large series took too long: %v", elapsed)
	}
	_ = result
}
