package handlers

import (
	"net/http"
	"strconv"

	"ancient-battlefield/pkg/battle_replay"
	"ancient-battlefield/pkg/battlefield_loader"
	"ancient-battlefield/pkg/config"
	"ancient-battlefield/pkg/defense_analyzer"
	"ancient-battlefield/pkg/doctrine_evolution"
	"ancient-battlefield/pkg/geo_partitioner"
	"ancient-battlefield/pkg/models"
	"ancient-battlefield/pkg/supply_analyzer"
	"ancient-battlefield/pkg/terrain_analyzer"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Cfg           *config.ModelConfig
	Loader        *battlefield_loader.Loader
	Analyzer      *terrain_analyzer.Analyzer
	Partitioner   *geo_partitioner.Partitioner
	Replay        *battle_replay.ReplayAnalyzer
	Supply        *supply_analyzer.SupplyAnalyzer
	Defense       *defense_analyzer.DefenseAnalyzer
	Doctrine      *doctrine_evolution.DoctrineAnalyzer
}

func New(cfg *config.ModelConfig) *Handler {
	h := &Handler{
		Cfg:         cfg,
		Loader:      battlefield_loader.New(""),
		Analyzer:    terrain_analyzer.New(cfg),
		Partitioner: geo_partitioner.New(cfg),
		Replay:      battle_replay.New(cfg),
		Supply:      supply_analyzer.New(cfg),
		Defense:     defense_analyzer.New(cfg),
		Doctrine:    doctrine_evolution.New(cfg),
	}
	_ = h.Loader.Load()
	return h
}

func (h *Handler) GetBattlefields(c *gin.Context) {
	c.JSON(http.StatusOK, h.Loader.GetBattlefields())
}

func (h *Handler) GetBattlefieldByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	for _, bf := range h.Loader.GetBattlefields() {
		if bf.ID == id {
			c.JSON(http.StatusOK, bf)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

func (h *Handler) GetRoads(c *gin.Context) {
	c.JSON(http.StatusOK, h.Loader.GetRoads())
}

func (h *Handler) GetRivers(c *gin.Context) {
	c.JSON(http.StatusOK, h.Loader.GetRivers())
}

func (h *Handler) GetDEM(c *gin.Context) {
	c.JSON(http.StatusOK, h.Loader.GetDEM())
}

func (h *Handler) GetTerrainProfile(c *gin.Context) {
	sl, _ := strconv.ParseFloat(c.DefaultQuery("start_lng", "115"), 64)
	slat, _ := strconv.ParseFloat(c.DefaultQuery("start_lat", "34"), 64)
	el, _ := strconv.ParseFloat(c.DefaultQuery("end_lng", "117"), 64)
	elat, _ := strconv.ParseFloat(c.DefaultQuery("end_lat", "34"), 64)
	n, _ := strconv.Atoi(c.DefaultQuery("num_points", "50"))
	prof := h.Partitioner.GenerateTerrainProfile(h.Loader.GetDEM(), sl, slat, el, elat, n)
	c.JSON(http.StatusOK, prof)
}

func (h *Handler) GetAccessibility(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	for _, bf := range h.Loader.GetBattlefields() {
		if bf.ID == id {
			c.JSON(http.StatusOK, h.Partitioner.ComputeAccessibility(bf, h.Loader.GetRoads()))
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

func (h *Handler) GetSiteSelectionFactors(c *gin.Context) {
	bgType := c.DefaultQuery("background", h.Cfg.BackgroundSampling.Type)
	bsRuns, _ := strconv.Atoi(c.DefaultQuery("bootstrap", strconv.Itoa(h.Cfg.Bootstrap.Runs)))
	bfs := h.Loader.GetBattlefields()
	lr := h.Analyzer.TrainEnhancedLogisticRegression(bfs, bgType, bsRuns)
	factors := h.Analyzer.ComputeFactorsFromResult(lr)
	h.Analyzer.SetLastResult(&lr, factors)
	c.JSON(http.StatusOK, gin.H{
		"factors":       factors,
		"model_metrics": gin.H{
			"auc":             lr.AUC,
			"accuracy":        lr.Accuracy,
			"precision":       lr.Precision,
			"recall":          lr.Recall,
			"f1":              lr.F1,
			"background_type": lr.BackgroundType,
			"bootstrap_runs":  lr.BootstrapRuns,
		},
	})
}

func (h *Handler) GetEnhancedLR(c *gin.Context) {
	bgType := c.DefaultQuery("background", h.Cfg.BackgroundSampling.Type)
	bsRuns, _ := strconv.Atoi(c.DefaultQuery("bootstrap", strconv.Itoa(h.Cfg.Bootstrap.Runs)))
	bfs := h.Loader.GetBattlefields()
	lr := h.Analyzer.TrainEnhancedLogisticRegression(bfs, bgType, bsRuns)
	h.Analyzer.SetLastResult(&lr, nil)
	c.JSON(http.StatusOK, lr)
}

func (h *Handler) GetHighProbAreas(c *gin.Context) {
	bfs := h.Loader.GetBattlefields()
	bgType := c.DefaultQuery("background", h.Cfg.BackgroundSampling.Type)
	bsRuns, _ := strconv.Atoi(c.DefaultQuery("bootstrap", strconv.Itoa(h.Cfg.Bootstrap.Runs)))
	lr, _ := h.Analyzer.GetLast()
	if lr == nil {
		rlr := h.Analyzer.TrainEnhancedLogisticRegression(bfs, bgType, bsRuns)
		lr = &rlr
		h.Analyzer.SetLastResult(lr, nil)
	}
	areas := h.Analyzer.ComputeHighProbAreas(bfs, *lr)
	c.JSON(http.StatusOK, areas)
}

func (h *Handler) GetMilitaryRegions(c *gin.Context) {
	numRegions, _ := strconv.Atoi(c.DefaultQuery("num_regions", strconv.Itoa(h.Cfg.Clustering.DefaultK)))
	fuzzy, _ := strconv.ParseBool(c.DefaultQuery("fuzzy", "true"))
	bfs := h.Loader.GetBattlefields()
	var regions []models.MilitaryRegion
	var fcm *models.FuzzyClusterResult
	if fuzzy {
		regs, fres := h.Partitioner.GenerateMilitaryRegionsFCM(bfs, numRegions)
		regions = regs
		fcm = &fres
	} else {
		regs := h.generateHardRegions(bfs, numRegions)
		regions = regs
		fcm = nil
	}
	h.Partitioner.SetLast(regions, fcm)
	result := gin.H{"regions": regions}
	if fcm != nil {
		result["fcm_result"] = fcm
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetFuzzyCluster(c *gin.Context) {
	k, _ := strconv.Atoi(c.DefaultQuery("k", strconv.Itoa(h.Cfg.Clustering.DefaultK)))
	bfs := h.Loader.GetBattlefields()
	_, fcm := h.Partitioner.GenerateMilitaryRegionsFCM(bfs, k)
	c.JSON(http.StatusOK, fcm)
}

func (h *Handler) GetStatistics(c *gin.Context) {
	bfs := h.Loader.GetBattlefields()
	eraCount := make(map[string]int)
	terrainCount := make(map[string]int)
	totalTroops := int64(0)
	for _, bf := range bfs {
		eraCount[bf.Era]++
		terrainCount[bf.TerrainType]++
		totalTroops += int64(bf.TotalTroops)
	}
	c.JSON(http.StatusOK, gin.H{
		"total_battlefields": len(bfs),
		"era_distribution":   eraCount,
		"terrain_distribution": terrainCount,
		"total_troops":       totalTroops,
		"avg_troops":         float64(totalTroops) / float64(len(bfs)),
		"total_roads":        len(h.Loader.GetRoads()),
		"total_rivers":       len(h.Loader.GetRivers()),
	})
}

func (h *Handler) findBattlefield(id int64) *models.Battlefield {
	for _, bf := range h.Loader.GetBattlefields() {
		if bf.ID == id {
			b := bf
			return &b
		}
	}
	return nil
}

func (h *Handler) GetBattleReplay(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	bf := h.findBattlefield(id)
	if bf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	fps, _ := strconv.Atoi(c.DefaultQuery("fps", strconv.Itoa(h.Cfg.BattleReplay.DefaultFps)))
	result := h.Replay.GenerateBattleReplay(*bf, fps)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetBattleEvents(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	bf := h.findBattlefield(id)
	if bf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	events := h.Replay.NLPExtractEvents(*bf)
	c.JSON(http.StatusOK, gin.H{
		"battlefield_id": bf.ID,
		"battle_name":    bf.BattleName,
		"events":         events,
		"event_count":    len(events),
	})
}

func (h *Handler) GetSupplyAnalysis(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	bf := h.findBattlefield(id)
	if bf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	result := h.Supply.AnalyzeSupply(*bf, h.Loader.GetRoads())
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetDefenseEvaluation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	bf := h.findBattlefield(id)
	if bf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	viewshedKm, _ := strconv.ParseFloat(c.DefaultQuery("viewshed_km",
		strconv.FormatFloat(h.Cfg.DefenseEvaluation.DefaultViewshedKm, 'f', -1, 64)))
	results := h.Defense.EvaluateAll(*bf, viewshedKm)
	c.JSON(http.StatusOK, gin.H{
		"battlefield_id": bf.ID,
		"battle_name":    bf.BattleName,
		"evaluations":    results,
		"structure_count": len(results),
	})
}

func (h *Handler) GetMilitaryStructures(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	bf := h.findBattlefield(id)
	if bf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	structures := h.Defense.GenerateMilitaryStructures(*bf)
	c.JSON(http.StatusOK, gin.H{
		"battlefield_id": bf.ID,
		"battle_name":    bf.BattleName,
		"structures":     structures,
		"count":          len(structures),
	})
}

func (h *Handler) GetDoctrineEvolution(c *gin.Context) {
	bfs := h.Loader.GetBattlefields()
	result := h.Doctrine.AnalyzeEvolution(bfs)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) generateHardRegions(bfs []models.Battlefield, k int) []models.MilitaryRegion {
	regions, _ := h.Partitioner.GenerateMilitaryRegionsFCM(bfs, k)
	return regions
}
