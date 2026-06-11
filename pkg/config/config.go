package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ModelConfig struct {
	LogisticRegression LRConfig       `yaml:"logistic_regression"`
	Bootstrap          BootstrapConfig `yaml:"bootstrap"`
	BackgroundSampling BackgroundConfig `yaml:"background_sampling"`
	Clustering         ClusteringConfig `yaml:"clustering"`
	HighProbArea       HPConfig       `yaml:"high_prob_area"`
	TerrainProfile     TPConfig       `yaml:"terrain_profile"`
	Accessibility      AccessConfig   `yaml:"accessibility"`
	BattleReplay       ReplayConfig   `yaml:"battle_replay"`
	SupplyAnalysis     SupplyConfig   `yaml:"supply_analysis"`
	DefenseEvaluation  DefenseConfig  `yaml:"defense_evaluation"`
	DoctrineEvolution  EvolutionConfig `yaml:"doctrine_evolution"`
}

type ReplayConfig struct {
	DefaultFps   int     `yaml:"default_fps"`
	MinEvents    int     `yaml:"min_events_per_battle"`
	MaxEvents    int     `yaml:"max_events_per_battle"`
	NLPConfidenceThreshold float64 `yaml:"nlp_confidence_threshold"`
}

type SupplyConfig struct {
	NodesPerSide       int     `yaml:"nodes_per_side"`
	DefaultSpeedKmh    float64 `yaml:"default_speed_kmh"`
	WorkingHoursPerDay float64 `yaml:"working_hours_per_day"`
	BottleneckPct      float64 `yaml:"bottleneck_percentage"`
}

type DefenseConfig struct {
	DefaultViewshedKm  float64 `yaml:"default_viewshed_sample_km"`
	BlindZoneThreshold float64 `yaml:"blind_zone_visibility_threshold"`
	NumDirections      int     `yaml:"viewshed_directions"`
	StructureCountMin  int     `yaml:"structures_per_battle_min"`
	StructureCountMax  int     `yaml:"structures_per_battle_max"`
}

type EvolutionConfig struct {
	ChangePointWindow  int     `yaml:"change_point_window"`
	SampleAnimationYears int   `yaml:"animation_sample_count"`
	TimeSeriesStep     int     `yaml:"time_series_step_battles"`
	TimeSeriesWindow   int     `yaml:"time_series_window"`
}

type LRConfig struct {
	LearningRate float64 `yaml:"learning_rate"`
	Epochs       int     `yaml:"epochs"`
	Tolerance    float64 `yaml:"tolerance"`
}

type BootstrapConfig struct {
	Runs         int     `yaml:"runs"`
	Confidence   float64 `yaml:"confidence_level"`
}

type BackgroundConfig struct {
	Type            string  `yaml:"default_type"`
	KernelBandwidth float64 `yaml:"target_group_bandwidth_deg"`
	NumBackground   int     `yaml:"num_background_points"`
}

type ClusteringConfig struct {
	DefaultK      int     `yaml:"default_num_regions"`
	FCM_Fuzzifier float64 `yaml:"fcm_fuzzifier"`
	FCM_MaxIter   int     `yaml:"fcm_max_iter"`
	FCM_Eps       float64 `yaml:"fcm_convergence_eps"`
	KM_MaxIter    int     `yaml:"kmeans_max_iter"`
	TroopScale    float64 `yaml:"troops_scale_factor"`
}

type HPConfig struct {
	GridLngMin  float64 `yaml:"grid_lng_min"`
	GridLngMax  float64 `yaml:"grid_lng_max"`
	GridLatMin  float64 `yaml:"grid_lat_min"`
	GridLatMax  float64 `yaml:"grid_lat_max"`
	GridLngStep float64 `yaml:"grid_lng_step_deg"`
	GridLatStep float64 `yaml:"grid_lat_step_deg"`
	Threshold   float64 `yaml:"threshold"`
}

type TPConfig struct {
	DefaultPoints int     `yaml:"default_num_points"`
	DefaultWidth  float64 `yaml:"default_width_deg"`
}

type AccessConfig struct {
	DecayRate float64 `yaml:"decay_rate"`
}

var DefaultConfig = ModelConfig{
	LogisticRegression: LRConfig{
		LearningRate: 0.0001,
		Epochs:       5000,
		Tolerance:    1e-7,
	},
	Bootstrap: BootstrapConfig{
		Runs:       100,
		Confidence: 0.95,
	},
	BackgroundSampling: BackgroundConfig{
		Type:            "target_group",
		KernelBandwidth: 5.0,
		NumBackground:   1000,
	},
	Clustering: ClusteringConfig{
		DefaultK:      8,
		FCM_Fuzzifier: 2.0,
		FCM_MaxIter:   100,
		FCM_Eps:       0.0001,
		KM_MaxIter:    50,
		TroopScale:    10000.0,
	},
	HighProbArea: HPConfig{
		GridLngMin:  73.0,
		GridLngMax:  135.0,
		GridLatMin:  18.0,
		GridLatMax:  54.0,
		GridLngStep: 1.0,
		GridLatStep: 1.0,
		Threshold:   0.6,
	},
	TerrainProfile: TPConfig{
		DefaultPoints: 50,
		DefaultWidth:  2.0,
	},
	Accessibility: AccessConfig{
		DecayRate: 0.05,
	},
	BattleReplay: ReplayConfig{
		DefaultFps:   12,
		MinEvents:    5,
		MaxEvents:    10,
		NLPConfidenceThreshold: 0.6,
	},
	SupplyAnalysis: SupplyConfig{
		NodesPerSide:       4,
		DefaultSpeedKmh:    15.0,
		WorkingHoursPerDay: 12.0,
		BottleneckPct:      0.3,
	},
	DefenseEvaluation: DefenseConfig{
		DefaultViewshedKm:  8.0,
		BlindZoneThreshold: 0.5,
		NumDirections:      36,
		StructureCountMin:  3,
		StructureCountMax:  5,
	},
	DoctrineEvolution: EvolutionConfig{
		ChangePointWindow:  5,
		SampleAnimationYears: 24,
		TimeSeriesStep:     5,
		TimeSeriesWindow:   30,
	},
}

func Load(path string) (*ModelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig
			return &cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg ModelConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	return &cfg, nil
}

func SaveDefault(path string) error {
	data, err := yaml.Marshal(&DefaultConfig)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
