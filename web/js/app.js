const App = (function() {
    let selectedBattlefield = null;

    let state = {
        showBattlefields: true,
        showRoads: true,
        showRivers: true,
        showRegions: false,
        showHighProb: false,
        showSupply: false,
        showStructures: false,
        showBlindzones: false,
        filterEra: '',
        filterTerrain: '',
        minTroops: 0,
        useOffscreen: true,
        backgroundType: 'target_group',
        bootstrapRuns: 100
    };

    async function init() {
        AppData.setState(state);

        BattlefieldMap.init('map', 'battlefield-canvas', handleBattlefieldSelect, handleBattlefieldHover);

        bindEvents();
        initPerfInfo();

        await AppData.loadAll();

        renderEraLegend();
        renderTroopsLegend();

        BattlefieldMap.drawRoads(AppData.getRoads());
        BattlefieldMap.drawRivers(AppData.getRivers());

        applyFilters();
        renderStats();
        loadFactorAnalysis();

        startRenderLoop();
    }

    function startRenderLoop() {
        function frame() {
            BattlefieldMap.render();
            requestAnimationFrame(frame);
        }
        requestAnimationFrame(frame);
    }

    function initPerfInfo() {
        const perfStatus = document.getElementById('perf-status');
        if (Charts.isOffscreenCanvasSupported()) {
            perfStatus.innerHTML = '渲染模式: <span class="supported">OffscreenCanvas (硬件加速)</span>';
        } else {
            perfStatus.innerHTML = '渲染模式: <span class="unsupported">标准Canvas (兼容模式)</span>';
            document.getElementById('toggle-offscreen').checked = false;
            state.useOffscreen = false;
            AppData.setState(state);
        }
    }

    function handleBattlefieldSelect(bf) {
        selectedBattlefield = bf;
        if (bf) {
            showDetailPanel(bf);
        } else {
            closePanel();
        }
    }

    function handleBattlefieldHover(bf) {
        BattlefieldMap.setHovered(bf);
        BattlefieldMap.render();
    }

    function bindEvents() {
        document.getElementById('toggle-battlefields').addEventListener('change', (e) => {
            state.showBattlefields = e.target.checked;
            AppData.setState(state);
        });
        document.getElementById('toggle-roads').addEventListener('change', (e) => {
            state.showRoads = e.target.checked;
            AppData.setState(state);
            const map = BattlefieldMap.getMap();
            AppData.getRoads().forEach(() => {});
        });
        document.getElementById('toggle-rivers').addEventListener('change', (e) => {
            state.showRivers = e.target.checked;
            AppData.setState(state);
        });
        document.getElementById('toggle-regions').addEventListener('change', (e) => {
            state.showRegions = e.target.checked;
            AppData.setState(state);
            BattlefieldMap.toggleRegions(e.target.checked);
            if (e.target.checked && AppData.getRegions().length === 0) {
                analyzeRegions();
            }
        });
        document.getElementById('toggle-highprob').addEventListener('change', (e) => {
            state.showHighProb = e.target.checked;
            AppData.setState(state);
            BattlefieldMap.toggleHighProb(e.target.checked);
            if (e.target.checked && AppData.getHighProbAreas().length === 0) {
                analyzeHighProb();
            }
        });

        document.getElementById('filter-era').addEventListener('change', (e) => {
            state.filterEra = e.target.value;
            AppData.setState(state);
            applyFilters();
        });
        document.getElementById('filter-terrain').addEventListener('change', (e) => {
            state.filterTerrain = e.target.value;
            AppData.setState(state);
            applyFilters();
        });
        document.getElementById('filter-troops').addEventListener('input', (e) => {
            state.minTroops = parseInt(e.target.value) || 0;
            AppData.setState(state);
            document.getElementById('troops-value').textContent = formatTroops(state.minTroops);
            applyFilters();
        });

        document.getElementById('panel-close').addEventListener('click', closePanel);
        document.getElementById('btn-analyze').addEventListener('click', analyzeRegions);
        document.getElementById('btn-highprob').addEventListener('click', analyzeHighProb);

        document.getElementById('background-type').addEventListener('change', (e) => {
            state.backgroundType = e.target.value;
            AppData.setState(state);
            loadFactorAnalysis();
        });
        document.getElementById('bootstrap-runs').addEventListener('change', (e) => {
            state.bootstrapRuns = parseInt(e.target.value) || 100;
            AppData.setState(state);
            loadFactorAnalysis();
        });
        document.getElementById('toggle-offscreen').addEventListener('change', (e) => {
            state.useOffscreen = e.target.checked;
            AppData.setState(state);
            Charts.clearOffscreenCache();
            if (selectedBattlefield) {
                showDetailPanel(selectedBattlefield);
            }
        });

        document.getElementById('toggle-supply').addEventListener('change', (e) => {
            state.showSupply = e.target.checked;
            AppData.setState(state);
            if (selectedBattlefield) {
                const map = BattlefieldMap.getMap();
                LogisticsAnalyzer.toggle(map, e.target.checked);
            }
        });
        document.getElementById('toggle-structures').addEventListener('change', (e) => {
            state.showStructures = e.target.checked;
            AppData.setState(state);
            if (selectedBattlefield) {
                const map = BattlefieldMap.getMap();
                DefenseEvaluator.toggleStructures(map, e.target.checked);
            }
        });
        document.getElementById('toggle-blindzones').addEventListener('change', (e) => {
            state.showBlindzones = e.target.checked;
            AppData.setState(state);
            if (selectedBattlefield) {
                const map = BattlefieldMap.getMap();
                DefenseEvaluator.toggleBlindZones(map, e.target.checked);
            }
        });

        document.getElementById('btn-replay').addEventListener('click', () => {
            if (!selectedBattlefield) {
                alert('请先在地图上选择一个战场');
                return;
            }
            const map = BattlefieldMap.getMap();
            BattleReplayer.initUI(map, selectedBattlefield);
        });
        document.getElementById('btn-supply').addEventListener('click', () => {
            if (!selectedBattlefield) {
                alert('请先在地图上选择一个战场');
                return;
            }
            const map = BattlefieldMap.getMap();
            LogisticsAnalyzer.initUI(map, selectedBattlefield);
            state.showSupply = true;
            AppData.setState(state);
        });
        document.getElementById('btn-defense').addEventListener('click', () => {
            if (!selectedBattlefield) {
                alert('请先在地图上选择一个战场');
                return;
            }
            const map = BattlefieldMap.getMap();
            DefenseEvaluator.initUI(map, selectedBattlefield);
            state.showStructures = true;
            state.showBlindzones = true;
            AppData.setState(state);
        });
        document.getElementById('btn-doctrine').addEventListener('click', () => {
            const map = BattlefieldMap.getMap();
            TacticalEvolution.initUI(map);
        });
    }

    function applyFilters() {
        const all = AppData.getBattlefields();
        const filtered = all.filter(bf => {
            if (state.filterEra && bf.era !== state.filterEra) return false;
            if (state.filterTerrain && bf.terrain_type !== state.filterTerrain) return false;
            if (bf.total_troops < state.minTroops) return false;
            return true;
        });
        AppData.setFiltered(filtered);
    }

    function renderEraLegend() {
        const legend = document.getElementById('era-legend');
        const colors = BattlefieldMap.getEraColors();
        const names = BattlefieldMap.getEraNames();
        legend.innerHTML = names.map(n => `
            <div class="legend-item">
                <span class="legend-triangle" style="border-bottom-color:${colors[n]}"></span>
                <span>${n}</span>
            </div>
        `).join('');
    }

    function renderTroopsLegend() {
        const legend = document.getElementById('troops-legend');
        const levels = [
            { min: 10000, max: 50000, size: 6, label: '<5万' },
            { min: 50000, max: 100000, size: 10, label: '5-10万' },
            { min: 100000, max: 200000, size: 14, label: '10-20万' },
            { min: 200000, max: 500000, size: 18, label: '20-50万' },
            { min: 500000, max: Infinity, size: 22, label: '>50万' }
        ];
        legend.innerHTML = levels.map(l => `
            <div class="legend-item">
                <span class="legend-triangle" style="border-bottom-width:${l.size}px;border-left-width:${l.size*0.87}px;border-right-width:${l.size*0.87}px;margin-left:${22 - l.size*0.87}px"></span>
                <span>${l.label}</span>
            </div>
        `).join('');
    }

    function renderStats() {
        const all = AppData.getBattlefields();
        document.getElementById('stat-total').textContent = all.length;
        document.getElementById('stat-eras').textContent = new Set(all.map(b => b.era)).size;
        const totalT = all.reduce((s, b) => s + b.total_troops, 0);
        document.getElementById('stat-troops').textContent = formatTroops(totalT);
    }

    function formatTroops(n) {
        if (n >= 10000) return (n / 10000).toFixed(0) + '万';
        return n.toString();
    }

    function renderAccessibilityInfo(acc) {
        const container = document.getElementById('accessibility-info');
        if (!container) return;
        container.innerHTML = `
            <div class="access-score">
                <span class="access-label">交通可达性综合评分</span>
                <span class="access-value">${acc.access_level} (${acc.access_score.toFixed(2)})</span>
            </div>
            <div class="access-item">
                <span class="access-label">最近道路距离</span>
                <span class="access-value">${acc.nearest_road_dist.toFixed(1)} km</span>
            </div>
            <div class="access-item">
                <span class="access-label">交通连通指数</span>
                <span class="access-value">${acc.connectivity_index.toFixed(1)}</span>
            </div>
        `;
    }

    async function loadFactorAnalysis() {
        try {
            const bg = state.backgroundType;
            const bs = state.bootstrapRuns;
            const data = await fetchWithFallback(
                `/api/site_selection_factors?background=${bg}&bootstrap=${bs}`,
                () => generateMockEnhancedFactors(bg, bs)
            );
            let factors, modelMetrics;
            if (data.factors && data.model_metrics) {
                factors = data.factors;
                modelMetrics = data.model_metrics;
            } else {
                factors = data;
                modelMetrics = null;
            }
            AppData.setFactors(factors);
            if (modelMetrics) {
                renderModelMetrics(modelMetrics);
                document.getElementById('model-metrics').classList.remove('hidden');
            }
            renderFactorAnalysis(factors);
            drawCharts(factors);
        } catch (e) {
            const fallback = generateMockEnhancedFactors(state.backgroundType, state.bootstrapRuns);
            AppData.setFactors(fallback.factors);
            renderModelMetrics(fallback.model_metrics);
            document.getElementById('model-metrics').classList.remove('hidden');
            renderFactorAnalysis(fallback.factors);
            drawCharts(fallback.factors);
        }
    }

    function drawCharts(factors) {
        const all = AppData.getBattlefields();
        Charts.drawEraDistribution(document.getElementById('chart-era'), all);
        Charts.drawTerrainDistribution(document.getElementById('chart-terrain'), all);
    }

    function renderModelMetrics(metrics) {
        document.getElementById('metric-auc').textContent = (metrics.auc || 0).toFixed(3);
        document.getElementById('metric-accuracy').textContent = ((metrics.accuracy || 0) * 100).toFixed(1) + '%';
        document.getElementById('metric-f1').textContent = (metrics.f1 || 0).toFixed(3);
    }

    function generateMockEnhancedFactors(bg, bs) {
        return {
            factors: [
                { factor_name: '地形高程', contribution: 0.38, p_value: 0.012, odds_ratio: 1.004, stability_score: 0.92, ci95_lower: 1.001, ci95_upper: 1.007, std_err: 0.0015 },
                { factor_name: '交通可达性', contribution: 0.35, p_value: 0.023, odds_ratio: 0.945, stability_score: 0.88, ci95_lower: 0.902, ci95_upper: 0.988, std_err: 0.021 },
                { factor_name: '水源距离', contribution: 0.27, p_value: 0.045, odds_ratio: 0.972, stability_score: 0.76, ci95_lower: 0.948, ci95_upper: 0.996, std_err: 0.012 }
            ],
            model_metrics: {
                auc: 0.842,
                accuracy: 0.785,
                precision: 0.762,
                recall: 0.814,
                f1: 0.787,
                background_type: bg,
                bootstrap_runs: bs
            }
        };
    }

    function renderFactorAnalysis(factors) {
        const container = document.getElementById('factor-analysis');
        container.innerHTML = factors.map(f => {
            const stability = f.stability_score || 0;
            const stabilityClass = stability >= 0.85 ? 'stability-high' : (stability >= 0.7 ? 'stability-medium' : 'stability-low');
            const ci = f.ci95_upper && f.ci95_lower ? `
                <div class="ci-bar">
                    <div class="ci-tick" style="left:0"></div>
                    <div class="ci-tick" style="left:100%"></div>
                    <div class="ci-bar-fill" style="left:25%;right:25%"></div>
                </div>
                <div class="factor-meta" style="font-size:10px;margin-top:2px;color:#8b8b7a">
                    <span>95% CI: [${f.ci95_lower.toFixed(3)}, ${f.ci95_upper.toFixed(3)}]</span>
                </div>
            ` : '';
            const stabilityBadge = f.stability_score ? `
                <span class="stability-indicator ${stabilityClass}" title="Bootstrap稳定性: ${(stability * 100).toFixed(0)}%"></span>
            ` : '';
            const color = stability >= 0.85 ? '27ae60' : (stability >= 0.7 ? 'f1c40f' : 'e74c3c');
            return `
            <div class="factor-item">
                <div class="factor-name">
                    <span>${stabilityBadge}${f.factor_name}</span>
                    <span style="color:#d4af37">${(f.contribution * 100).toFixed(1)}%</span>
                </div>
                <div class="factor-bar">
                    <div class="factor-bar-fill" style="width:${f.contribution * 100}%"></div>
                </div>
                ${ci}
                <div class="factor-meta">
                    <span>P值: ${f.p_value.toFixed(3)}</span>
                    <span>OR: ${f.odds_ratio.toFixed(3)}</span>
                    ${f.stability_score ? `<span style="color:#${color}">稳定: ${(stability * 100).toFixed(0)}%</span>` : ''}
                </div>
            </div>
        `}).join('');
    }

    async function showDetailPanel(bf) {
        const panel = document.getElementById('detail-panel');
        panel.classList.add('active');

        document.getElementById('detail-name').textContent = bf.battle_name;
        document.getElementById('detail-era').textContent = bf.era;
        document.getElementById('detail-dynasty').textContent = bf.dynasty || '-';
        document.getElementById('detail-year').textContent = formatYear(bf.battle_year);
        document.getElementById('detail-belligerents').textContent = `${bf.belligerent_a} vs ${bf.belligerent_b}`;
        document.getElementById('detail-troops').textContent = `${formatTroops(bf.troop_a)} vs ${formatTroops(bf.troop_b)} (共${formatTroops(bf.total_troops)})`;
        document.getElementById('detail-terrain').textContent = bf.terrain_type;
        document.getElementById('detail-elevation').textContent = bf.elevation + ' m';
        document.getElementById('detail-result').textContent = bf.result;

        const bEventsDiv = document.getElementById('battle-events');
        if (bEventsDiv) {
            BattleReplayer.getEventsForBattle(bf.id, bf, function(data) {
                if (data && data.events && data.events.length) {
                    bEventsDiv.innerHTML = data.events.map(function(ev) {
                        var cls = 'event-item';
                        if (ev.is_turning_point) cls += ' turning-point';
                        if (ev.is_decision) cls += ' decision';
                        return '<div class="' + cls + '">' +
                            '<span class="event-time">' + BattleReplayer.formatHour(ev.hour_offset) + '</span>' +
                            '<span class="event-name">' + ev.event_name + '</span>' +
                            '<span class="event-type-tag">' + ev.event_type + '</span>' +
                            (ev.is_turning_point ? '<span class="event-type-tag" style="background:#e74c3c;color:#fff">转折点</span>' : '') +
                            (ev.is_decision ? '<span class="event-type-tag" style="background:#3498db;color:#fff">决策</span>' : '') +
                            '</div>';
                    }).join('');
                }
            });
        }

        try {
            const acc = await fetchWithFallback(
                `/api/accessibility/${bf.id}`,
                () => generateMockAccessibility(bf)
            );
            renderAccessibilityInfo(acc);
        } catch (e) {
            renderAccessibilityInfo(generateMockAccessibility(bf));
        }

        try {
            const profile = await fetchWithFallback(
                `/api/terrain_profile?start_lng=${bf.lng - 1}&start_lat=${bf.lat}&end_lng=${bf.lng + 1}&end_lat=${bf.lat}&num_points=50`,
                () => generateMockProfile(bf)
            );
            Charts.drawTerrainProfile(document.getElementById('profile-canvas'), profile, { useOffscreen: state.useOffscreen });
        } catch (e) {
            Charts.drawTerrainProfile(document.getElementById('profile-canvas'), generateMockProfile(bf), { useOffscreen: state.useOffscreen });
        }
    }

    function closePanel() {
        document.getElementById('detail-panel').classList.remove('active');
        const map = BattlefieldMap.getMap();
        BattleReplayer.reset(map);
        LogisticsAnalyzer.reset(map);
        DefenseEvaluator.reset(map);
        state.showSupply = false;
        state.showStructures = false;
        state.showBlindzones = false;
        AppData.setState(state);
        document.getElementById('toggle-supply').checked = false;
        document.getElementById('toggle-structures').checked = false;
        document.getElementById('toggle-blindzones').checked = false;
        selectedBattlefield = null;
        BattlefieldMap.setSelected(null);
    }

    function formatYear(y) {
        if (y < 0) return `公元前${-y}年`;
        return `公元${y}年`;
    }

    function generateMockAccessibility(bf) {
        const dist = 10 + Math.random() * 50;
        const score = 1 / (1 + dist / 30);
        return {
            nearest_road_dist: dist,
            access_score: score,
            access_level: score > 0.6 ? '高' : (score > 0.3 ? '中' : '低'),
            connectivity_index: 10 + Math.random() * 50
        };
    }

    function generateMockProfile(bf) {
        const n = 50;
        const base = bf.elevation;
        const points = [];
        let minE = Infinity, maxE = -Infinity, sumE = 0;
        for (let i = 0; i < n; i++) {
            const t = i / (n - 1);
            const noise = Math.sin(t * 8) * 100 + Math.cos(t * 13) * 60;
            const trend = (t - 0.5) * base * 0.3;
            const e = Math.max(0, base + noise + trend);
            minE = Math.min(minE, e);
            maxE = Math.max(maxE, e);
            sumE += e;
            points.push({
                lng: bf.lng - 1 + t * 2,
                lat: bf.lat,
                distance: t * 222,
                elevation: e
            });
        }
        return {
            start_lng: bf.lng - 1, start_lat: bf.lat,
            end_lng: bf.lng + 1, end_lat: bf.lat,
            num_points: n, min_elev: minE, max_elev: maxE,
            avg_elev: sumE / n, total_dist: 222, points
        };
    }

    async function analyzeRegions() {
        const btn = document.getElementById('btn-analyze');
        btn.textContent = '分析中...';
        btn.disabled = true;

        try {
            const data = await fetchWithFallback(
                '/api/military_regions?num_regions=8&fuzzy=true',
                () => generateMockRegionsFCM()
            );
            let regions, fcmResult;
            if (data.regions && data.fcm_result) {
                regions = data.regions;
                fcmResult = data.fcm_result;
            } else {
                regions = data;
                fcmResult = null;
            }
            AppData.setRegions(regions);
            if (fcmResult) {
                renderClusterQuality(fcmResult);
                document.getElementById('cluster-uncertainty').classList.remove('hidden');
            }
            BattlefieldMap.drawRegionsWithUncertainty(regions, fcmResult);
            document.getElementById('stat-regions').textContent = regions.length;
            document.getElementById('toggle-regions').checked = true;
            state.showRegions = true;
            AppData.setState(state);
        } finally {
            btn.textContent = '运行军事地理分区';
            btn.disabled = false;
        }
    }

    function renderClusterQuality(fcm) {
        document.getElementById('metric-pc').textContent = (fcm.partition_coef || 0).toFixed(3);
        document.getElementById('metric-pe').textContent = (fcm.partition_entropy || 0).toFixed(3);
        const avgUnc = fcm.uncertainties ? fcm.uncertainties.reduce((a, b) => a + b, 0) / fcm.uncertainties.length : 0;
        document.getElementById('metric-avg-uncertainty').textContent = (avgUnc * 100).toFixed(1) + '%';
    }

    function generateMockRegionsFCM() {
        const mockRegions = generateMockRegions();
        const n = mockRegions.length;
        const uncertainties = new Array(n).fill(0).map(() => 0.05 + Math.random() * 0.25);
        return {
            regions: mockRegions.map((r, i) => ({
                ...r,
                avg_membership: 0.75 + Math.random() * 0.2,
                uncertainty: uncertainties[i]
            })),
            fcm_result: {
                partition_coef: 0.68 + Math.random() * 0.25,
                partition_entropy: 0.15 + Math.random() * 0.3,
                uncertainties: uncertainties,
                membership_matrix: new Array(n).fill(0).map(() =>
                    new Array(8).fill(0).map(() => Math.random()).map((v, i, arr) => v / arr.reduce((a, b) => a + b, 0))
                )
            }
        };
    }

    function generateMockRegions() {
        const centers = [
            { lng: 114, lat: 34, name: '中原军区' },
            { lng: 108, lat: 34, name: '关中军区' },
            { lng: 118, lat: 32, name: '江淮军区' },
            { lng: 120, lat: 40, name: '幽燕军区' },
            { lng: 103, lat: 36, name: '河西军区' },
            { lng: 104, lat: 30, name: '巴蜀军区' },
            { lng: 112, lat: 30, name: '荆州军区' },
            { lng: 110, lat: 25, name: '岭南军区' }
        ];
        const terrains = ['山地', '平原', '河谷', '关隘'];
        return centers.map((c, i) => {
            const r = 2 + Math.random() * 1.5;
            const coords = [];
            for (let v = 0; v < 20; v++) {
                const theta = v * 2 * Math.PI / 20;
                coords.push([c.lng + r * Math.cos(theta), c.lat + r * Math.sin(theta)]);
            }
            return {
                id: i + 1,
                region_code: `MR-${i + 1}`,
                region_name: c.name,
                battle_count: 50 + Math.floor(Math.random() * 150),
                avg_density: 2 + Math.random() * 8,
                dominant_terrain: terrains[Math.floor(Math.random() * 4)],
                center_lng: c.lng,
                center_lat: c.lat,
                coords: [coords]
            };
        });
    }

    async function analyzeHighProb() {
        const btn = document.getElementById('btn-highprob');
        btn.textContent = '分析中...';
        btn.disabled = true;
        try {
            const areas = await fetchWithFallback(
                `/api/high_prob_areas?background=${state.backgroundType}&bootstrap=${state.bootstrapRuns}`,
                () => generateMockHighProb()
            );
            AppData.setHighProbAreas(areas);
            BattlefieldMap.drawHighProbAreas(areas);
            document.getElementById('toggle-highprob').checked = true;
            state.showHighProb = true;
            AppData.setState(state);
        } finally {
            btn.textContent = '分析高概率区域';
            btn.disabled = false;
        }
    }

    function generateMockHighProb() {
        const centers = [
            { lng: 114, lat: 34, name: '中原高概率区', p: 0.85 },
            { lng: 108, lat: 30, name: '巴蜀高概率区', p: 0.72 },
            { lng: 118, lat: 31, name: '江东高概率区', p: 0.78 },
            { lng: 116, lat: 39, name: '幽燕高概率区', p: 0.69 },
            { lng: 103, lat: 36, name: '河西高概率区', p: 0.63 }
        ];
        return centers.map((c, i) => {
            const r = 2.5;
            return {
                id: i + 1,
                area_name: c.name,
                probability: c.p,
                coords: [[
                    [c.lng - r, c.lat - r], [c.lng + r, c.lat - r],
                    [c.lng + r, c.lat + r], [c.lng - r, c.lat + r]
                ]]
            };
        });
    }

    async function fetchWithFallback(url, fallbackFn) {
        try {
            const res = await fetch(url);
            if (res.ok) return await res.json();
        } catch (e) {}
        return typeof fallbackFn === 'function' ? fallbackFn() : null;
    }

    return { init };
})();

document.addEventListener('DOMContentLoaded', App.init);
