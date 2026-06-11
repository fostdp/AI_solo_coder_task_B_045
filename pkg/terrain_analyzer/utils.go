package terrain_analyzer

import (
	"math"
	"sync"
)

type RNG struct {
	mu sync.Mutex
	s  uint64
}

type lockingRng struct {
	mu sync.Mutex
	s  uint64
}

func NewRNG(seed uint64) *RNG {
	if seed == 0 {
		seed = 42
	}
	return &RNG{s: seed}
}

func (r *RNG) Intn(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 {
		return 0
	}
	r.s = r.s*6364136223846793005 + 1442695040888963407
	v := int(r.s >> 33)
	if v < 0 {
		v = -v
	}
	return v % n
}

func (r *RNG) Float64() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.s = r.s*6364136223846793005 + 1442695040888963407
	return float64(r.s>>11) / (1 << 53)
}

func (l *lockingRng) Float64() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.s == 0 {
		l.s = 2024
	}
	l.s = l.s*6364136223846793005 + 1442695040888963407
	return float64(l.s>>11) / (1 << 53)
}

func (l *lockingRng) Intn(n int) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.s == 0 {
		l.s = 2024
	}
	if n <= 0 {
		return 0
	}
	l.s = l.s*6364136223846793005 + 1442695040888963407
	v := int(l.s >> 33)
	if v < 0 {
		v = -v
	}
	return v % n
}

var _ = math.Abs
