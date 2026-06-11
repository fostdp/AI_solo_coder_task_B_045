package models

type Battlefield struct {
	ID             int      `json:"id"`
	BattleName     string   `json:"battle_name"`
	Dynasty        string   `json:"dynasty"`
	Era            string   `json:"era"`
	BattleYear     int      `json:"battle_year"`
	BelligerentA   string   `json:"belligerent_a"`
	BelligerentB   string   `json:"belligerent_b"`
	TroopA         int      `json:"troop_a"`
	TroopB         int      `json:"troop_b"`
	TotalTroops    int      `json:"total_troops"`
	TerrainType    string   `json:"terrain_type"`
	Result         string   `json:"result"`
	Lng            float64  `json:"lng"`
	Lat            float64  `json:"lat"`
	Elevation      float64  `json:"elevation"`
	DistanceToRiver float64 `json:"distance_to_river"`
	DistanceToRoad  float64 `json:"distance_to_road"`
}

type AncientRoad struct {
	ID         int       `json:"id"`
	RoadName   string    `json:"road_name"`
	RoadType   string    `json:"road_type"`
	Dynasty    string    `json:"dynasty"`
	Importance int       `json:"importance"`
	Coords     [][2]float64 `json:"coords"`
}

type AncientRiver struct {
	ID        int       `json:"id"`
	RiverName string    `json:"river_name"`
	RiverType string    `json:"river_type"`
	Coords    [][2]float64 `json:"coords"`
}

type MilitaryRegion struct {
	ID             int       `json:"id"`
	RegionName     string    `json:"region_name"`
	RegionCode     string    `json:"region_code"`
	BattleCount    int       `json:"battle_count"`
	AvgDensity     float64   `json:"avg_density"`
	DominantTerrain string   `json:"dominant_terrain"`
	Coords         [][][2]float64 `json:"coords"`
	AvgMembership  float64   `json:"avg_membership"`
	Uncertainty   float64   `json:"uncertainty"`
	PartitionCoef float64   `json:"partition_coef"`
	Entropy       float64   `json:"entropy"`
}

type FuzzyClusterResult struct {
	Centroids     [][]float64   `json:"centroids"`
	Membership    [][]float64   `json:"membership"`
	Labels        []int         `json:"labels"`
	PartitionCoef float64       `json:"partition_coef"`
	PartitionEnt  float64       `json:"partition_entropy"`
	AvgUncertainty float64      `json:"avg_uncertainty"`
	Fuzzifier     float64       `json:"fuzzifier"`
	PointUncertainty []float64  `json:"point_uncertainty"`
	PointMembership []float64   `json:"point_membership"`
}

type HighProbArea struct {
	ID            int        `json:"id"`
	Probability   float64    `json:"probability"`
	TerrainFactor float64    `json:"terrain_factor"`
	RoadFactor    float64    `json:"road_factor"`
	RiverFactor   float64    `json:"river_factor"`
	Coords        [][][2]float64 `json:"coords"`
}

type SiteSelectionFactor struct {
	ID             int     `json:"id"`
	FactorName     string  `json:"factor_name"`
	Contribution   float64 `json:"contribution"`
	PValue         float64 `json:"p_value"`
	OddsRatio      float64 `json:"odds_ratio"`
	Method         string  `json:"method"`
	StdErr         float64 `json:"std_err"`
	CI95Lower      float64 `json:"ci95_lower"`
	CI95Upper      float64 `json:"ci95_upper"`
	Significance   bool    `json:"significance"`
	StabilityScore float64 `json:"stability_score"`
}

type EnhancedLRResult struct {
	Intercept     float64   `json:"intercept"`
	Coefficients  []float64 `json:"coefficients"`
	FactorNames   []string  `json:"factor_names"`
	Contributions []float64 `json:"contributions"`
	PValues       []float64 `json:"p_values"`
	OddsRatios    []float64 `json:"odds_ratios"`
	StdErrs       []float64 `json:"std_errs"`
	CI95Lowers    []float64 `json:"ci95_lowers"`
	CI95Uppers    []float64 `json:"ci95_uppers"`
	Stability     []float64 `json:"stability"`
	BootstrapRuns int       `json:"bootstrap_runs"`
	AUC           float64   `json:"auc"`
	Accuracy      float64   `json:"accuracy"`
	Precision     float64   `json:"precision"`
	Recall        float64   `json:"recall"`
	F1Score       float64   `json:"f1_score"`
	BackgroundType string   `json:"background_type"`
	NumBackground int       `json:"num_background"`
}

type ProfilePoint struct {
	Distance  float64 `json:"distance"`
	Elevation float64 `json:"elevation"`
}

type TerrainProfile struct {
	StartLng  float64        `json:"start_lng"`
	StartLat  float64        `json:"start_lat"`
	EndLng    float64        `json:"end_lng"`
	EndLat    float64        `json:"end_lat"`
	MinElev   float64        `json:"min_elev"`
	MaxElev   float64        `json:"max_elev"`
	AvgElev   float64        `json:"avg_elev"`
	Points    []ProfilePoint `json:"points"`
}

type AccessibilityAnalysis struct {
	BattlefieldID   int      `json:"battlefield_id"`
	NearestRoadDist float64  `json:"nearest_road_dist"`
	NearestRoadName string   `json:"nearest_road_name"`
	NearestRiverDist float64 `json:"nearest_river_dist"`
	NearestRiverName string   `json:"nearest_river_name"`
	RoadCountIn10km int      `json:"road_count_in_10km"`
	RiverCountIn10km int     `json:"river_count_in_10km"`
	AccessibilityScore float64 `json:"accessibility_score"`
}

type StatsByEra struct {
	Era      string `json:"era"`
	Count    int    `json:"count"`
	AvgTroops float64 `json:"avg_troops"`
}

type StatsByTerrain struct {
	TerrainType string  `json:"terrain_type"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
}

type DEMTile struct {
	ID         int      `json:"id"`
	TileX      int      `json:"tile_x"`
	TileY      int      `json:"tile_y"`
	TileZ      int      `json:"tile_z"`
	MinElev    int      `json:"min_elev"`
	MaxElev    int      `json:"max_elev"`
	GridSize   int      `json:"grid_size"`
	HeightGrid [][]int  `json:"height_grid"`
}

type BattleEvent struct {
	ID             int         `json:"id"`
	BattlefieldID  int         `json:"battlefield_id"`
	EventOrder     int         `json:"event_order"`
	EventType      string      `json:"event_type"`
	EventName      string      `json:"event_name"`
	Description    string      `json:"description"`
	HourOffset     float64     `json:"hour_offset"`
	Lng            float64     `json:"lng"`
	Lat            float64     `json:"lat"`
	Belligerent    string      `json:"belligerent"`
	TroopCount     int         `json:"troop_count"`
	Casualties     int         `json:"casualties"`
	IsTurningPoint bool        `json:"is_turning_point"`
	IsDecision     bool        `json:"is_decision"`
	Tags           []string    `json:"tags"`
	ExtractedFrom  string      `json:"extracted_from"`
	NLPConfidence  float64     `json:"nlp_confidence"`
}

type CampaignTimeline struct {
	BattlefieldID  int           `json:"battlefield_id"`
	BattleName     string        `json:"battle_name"`
	TotalDurationH float64       `json:"total_duration_h"`
	Events         []BattleEvent `json:"events"`
	TurningPoints  []BattleEvent `json:"turning_points"`
	Decisions      []BattleEvent `json:"decisions"`
}

type AnimationFrame struct {
	FrameIndex     int       `json:"frame_index"`
	TimestampH     float64   `json:"timestamp_h"`
	TimeLabel      string    `json:"time_label"`
	TroopPositions []struct {
		Belligerent  string    `json:"belligerent"`
		UnitName     string    `json:"unit_name"`
		Lng          float64   `json:"lng"`
		Lat          float64   `json:"lat"`
		TroopCount   int       `json:"troop_count"`
		IconType     string    `json:"icon_type"`
	} `json:"troop_positions"`
	ActiveEvent    *BattleEvent `json:"active_event,omitempty"`
	FrontLines     [][][2]float64 `json:"front_lines"`
}

type BattleReplayResult struct {
	Timeline       CampaignTimeline  `json:"timeline"`
	Frames         []AnimationFrame  `json:"frames"`
	Fps            int               `json:"fps"`
	TotalFrames    int               `json:"total_frames"`
	NLPStats       struct {
		TotalEvents      int     `json:"total_events"`
		AvgConfidence    float64 `json:"avg_confidence"`
		TurningPointCount int    `json:"turning_point_count"`
		DecisionCount    int     `json:"decision_count"`
	} `json:"nlp_stats"`
}

type SupplyNode struct {
	ID             int       `json:"id"`
	NodeName       string    `json:"node_name"`
	NodeType       string    `json:"node_type"`
	Belligerent    string    `json:"belligerent"`
	Lng            float64   `json:"lng"`
	Lat            float64   `json:"lat"`
	Capacity       int       `json:"capacity"`
	IsBottleneck   bool      `json:"is_bottleneck"`
	Throughput     float64   `json:"throughput"`
	RoadIDs        []int     `json:"road_ids"`
}

type SupplyRoute struct {
	ID             int         `json:"id"`
	RouteName      string      `json:"route_name"`
	Belligerent    string      `json:"belligerent"`
	Coords         [][2]float64 `json:"coords"`
	RoadSegments   []int       `json:"road_segments"`
	TotalLengthKm  float64     `json:"total_length_km"`
	Capacity       int         `json:"capacity"`
	EstTimeDays    float64     `json:"est_time_days"`
	Efficiency     float64     `json:"efficiency"`
	Nodes          []int       `json:"nodes"`
	BottleneckIDs  []int       `json:"bottleneck_ids"`
}

type SupplyAnalysis struct {
	BattlefieldID  int                `json:"battlefield_id"`
	BelligerentA   string             `json:"belligerent_a"`
	BelligerentB   string             `json:"belligerent_b"`
	RoutesA        []SupplyRoute      `json:"routes_a"`
	RoutesB        []SupplyRoute      `json:"routes_b"`
	NodesA         []SupplyNode       `json:"nodes_a"`
	NodesB         []SupplyNode       `json:"nodes_b"`
	BottlenecksA   []SupplyNode       `json:"bottlenecks_a"`
	BottlenecksB   []SupplyNode       `json:"bottlenecks_b"`
	AdvantageSide  string             `json:"advantage_side"`
	AdvantageScore float64            `json:"advantage_score"`
}

type MilitaryStructure struct {
	ID             int         `json:"id"`
	StructureName  string      `json:"structure_name"`
	StructureType  string      `json:"structure_type"`
	BattlefieldID  int         `json:"battlefield_id"`
	Dynasty        string      `json:"dynasty"`
	Lng            float64     `json:"lng"`
	Lat            float64     `json:"lat"`
	HeightM        float64     `json:"height_m"`
	LengthM        float64     `json:"length_m"`
	ThicknessM     float64     `json:"thickness_m"`
	Material       string      `json:"material"`
	GateCount      int         `json:"gate_count"`
	TowerCount     int         `json:"tower_count"`
	Coords         [][][2]float64 `json:"coords"`
}

type DefenseBlindZone struct {
	ID             int         `json:"id"`
	StructureID    int         `json:"structure_id"`
	CenterLng      float64     `json:"center_lng"`
	CenterLat      float64     `json:"center_lat"`
	AreaKm2        float64     `json:"area_km2"`
	Direction      string      `json:"direction"`
	MaxDistanceKm  float64     `json:"max_distance_km"`
	VisibilityPct  float64     `json:"visibility_pct"`
	RiskLevel      string      `json:"risk_level"`
	Coords         [][][2]float64 `json:"coords"`
}

type DefenseEvaluation struct {
	StructureID       int                    `json:"structure_id"`
	StructureName     string                 `json:"structure_name"`
	OverallScore      float64                `json:"overall_score"`
	VisibilityScore   float64                `json:"visibility_score"`
	StructuralScore   float64                `json:"structural_score"`
	TopographicScore  float64                `json:"topographic_score"`
	BlindZoneCount    int                    `json:"blind_zone_count"`
	TotalBlindAreaKm2 float64                `json:"total_blind_area_km2"`
	AvgVisibilityPct  float64                `json:"avg_visibility_pct"`
	BlindZones        []DefenseBlindZone     `json:"blind_zones"`
	ViewshedSampleKm  float64                `json:"viewshed_sample_km"`
	Recommendations   []string               `json:"recommendations"`
}

type EraDoctrineProfile struct {
	Era            string    `json:"era"`
	YearRange      [2]int    `json:"year_range"`
	BattleCount    int       `json:"battle_count"`
	AvgElevation   float64   `json:"avg_elevation"`
	AvgDistToRoad  float64   `json:"avg_dist_to_road"`
	AvgDistToRiver float64   `json:"avg_dist_to_river"`
	AvgTroops      float64   `json:"avg_troops"`
	TerrainDist    map[string]float64 `json:"terrain_dist"`
	DominantTerrain string   `json:"dominant_terrain"`
	DoctrineTag    string    `json:"doctrine_tag"`
	Characteristic string    `json:"characteristic"`
}

type ChangePoint struct {
	ID             int       `json:"id"`
	Year           int       `json:"year"`
	EraBoundary    string    `json:"era_boundary"`
	BeforeDoctrine string    `json:"before_doctrine"`
	AfterDoctrine  string    `json:"after_doctrine"`
	ChangeMagnitude float64  `json:"change_magnitude"`
	Confidence     float64   `json:"confidence"`
	KeyFeatures    []string  `json:"key_features"`
	TriggerEvents  []string  `json:"trigger_events"`
}

type DoctrineEvolutionResult struct {
	Profiles       []EraDoctrineProfile `json:"profiles"`
	ChangePoints   []ChangePoint        `json:"change_points"`
	TimeAnimation  []struct {
		Year           int                 `json:"year"`
		Era            string              `json:"era"`
		DoctrineTag    string              `json:"doctrine_tag"`
		HotspotCenters [][2]float64        `json:"hotspot_centers"`
		HeatmapData    []struct {
			Lng   float64 `json:"lng"`
			Lat   float64 `json:"lat"`
			Value float64 `json:"value"`
		} `json:"heatmap_data"`
		Features       map[string]float64  `json:"features"`
	} `json:"time_animation"`
	TimeSeries     []struct {
		Year      int     `json:"year"`
		Elevation float64 `json:"elevation"`
		Troops    float64 `json:"troops"`
		RoadDist  float64 `json:"road_dist"`
		Mobility  float64 `json:"mobility_index"`
	} `json:"time_series"`
	SummaryTrends  map[string]string  `json:"summary_trends"`
}
