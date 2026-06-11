var TacticalEvolution = (function () {
    var state = {
        data: null,
        currentFrame: 0,
        playing: false,
        timer: null,
        fps: 4,
        evolutionLayers: [],
        hotspotMarkers: []
    };

    var ERA_LIST = [
        { era: '春秋战国', year_range: [-770, -221], doctrine: '车战为主', characteristic: '战车为核心，依托平原列阵，讲究堂堂之阵', color: '#e74c3c' },
        { era: '秦汉', year_range: [-221, 220], doctrine: '骑兵崛起', characteristic: '骑兵成独立兵种，长途奔袭与步骑协同', color: '#e67e22' },
        { era: '三国两晋南北朝', year_range: [220, 589], doctrine: '骑步协同', characteristic: '重甲骑兵突击，水战与城防并重', color: '#f1c40f' },
        { era: '隋唐五代', year_range: [589, 960], doctrine: '步骑协同', characteristic: '府兵制与骑兵结合，野战与攻城并重', color: '#27ae60' },
        { era: '宋辽金元', year_range: [960, 1368], doctrine: '以步制骑/骑兵巅峰', characteristic: '宋军以步制骑防御，蒙元骑兵机动作战巅峰', color: '#3498db' },
        { era: '明清', year_range: [1368, 1912], doctrine: '火器时代', characteristic: '火器普遍装备，冷热兵器协同，坚城防御', color: '#9b59b6' }
    ];

    function fetchWithFallback(url, fallbackFn, callback) {
        fetch(url).then(function (r) {
            if (!r.ok) throw new Error('fail');
            return r.json();
        }).then(callback).catch(function () {
            if (fallbackFn) callback(fallbackFn());
        });
    }

    function formatYear(y) {
        if (y < 0) return '公元前' + (-y) + '年';
        return '公元' + y + '年';
    }

    function mockEvolutionData() {
        var battlefields = AppData.getBattlefields();
        var profiles = ERA_LIST.map(function (e) {
            var battles = battlefields.filter(function (b) {
                return b.battle_year >= e.year_range[0] && b.battle_year < e.year_range[1];
            });
            var avgElev = battles.length ? battles.reduce(function (s, b) { return s + b.elevation; }, 0) / battles.length : 500;
            var avgTroops = battles.length ? battles.reduce(function (s, b) { return s + b.total_troops; }, 0) / battles.length : 50000;
            var terrainDist = { '山地': 0.25, '平原': 0.3, '河谷': 0.25, '关隘': 0.2 };
            return {
                era: e.era,
                year_range: e.year_range,
                battle_count: battles.length,
                avg_elevation: avgElev,
                avg_troops: avgTroops,
                terrain_distribution: terrainDist,
                doctrine_tag: e.doctrine,
                characteristic: e.characteristic
            };
        });

        var changePoints = [];
        for (var i = 0; i < ERA_LIST.length - 1; i++) {
            var before = ERA_LIST[i];
            var after = ERA_LIST[i + 1];
            changePoints.push({
                id: i + 1,
                year: after.year_range[0],
                era_boundary: before.era + '→' + after.era,
                before_doctrine: before.doctrine,
                after_doctrine: after.doctrine,
                change_magnitude: 0.4 + Math.random() * 0.4,
                confidence: 0.75 + Math.random() * 0.2,
                key_features: ['平均海拔变化', '兵力规模变化', '地形偏好偏移'],
                trigger_events: [before.era + '末期战乱', after.era + '军事改革']
            });
        }

        var timeAnim = [];
        var totalSteps = 24;
        var minYear = -770, maxYear = 1912;
        for (var s = 0; s < totalSteps; s++) {
            var t = s / (totalSteps - 1);
            var year = Math.round(minYear + t * (maxYear - minYear));
            var eraIdx = 0;
            for (var ei = 0; ei < ERA_LIST.length; ei++) {
                if (year >= ERA_LIST[ei].year_range[0] && year < ERA_LIST[ei].year_range[1]) {
                    eraIdx = ei;
                    break;
                }
            }
            var era = ERA_LIST[eraIdx];
            var hotspots = [];
            var hotspotCount = 3 + Math.floor(Math.random() * 4);
            for (var h = 0; h < hotspotCount; h++) {
                hotspots.push({
                    lng: 95 + Math.random() * 40,
                    lat: 25 + Math.random() * 20,
                    intensity: 0.3 + Math.random() * 0.7
                });
            }
            var samples = [];
            for (var sp = 0; sp < 60; sp++) {
                samples.push({
                    lng: 73 + Math.random() * 62,
                    lat: 18 + Math.random() * 36,
                    weight: Math.random() * (0.5 + t * 0.5)
                });
            }
            timeAnim.push({
                frame_index: s,
                year: year,
                era: era.era,
                doctrine: era.doctrine,
                era_color: era.color,
                hotspots: hotspots,
                heat_samples: samples,
                feature_vector: {
                    avg_elevation: 300 + t * 600 + Math.random() * 100,
                    avg_troops: 30000 + t * 120000 + Math.random() * 20000,
                    mountain_ratio: 0.2 + Math.random() * 0.1,
                    river_proximity: 10 + Math.random() * 20,
                    road_proximity: 5 + Math.random() * 15
                }
            });
        }

        var timeSeries = [];
        for (var ts = 0; ts < 50; ts++) {
            var tts = ts / 49;
            var y = Math.round(minYear + tts * (maxYear - minYear));
            timeSeries.push({
                year: y,
                avg_elevation: 400 + tts * 500 + Math.sin(tts * 10) * 80,
                avg_troops: 40000 + tts * 120000 + Math.sin(tts * 8) * 15000,
                battle_count: 10 + Math.floor(Math.random() * 30),
                mountain_ratio: 0.2 + Math.random() * 0.15
            });
        }

        var summaryTrends = [
            { dimension: '战场海拔', trend: '逐步升高', description: '从春秋中原低地作战，发展到唐宋明清的山地要塞争夺，平均海拔上升约600米' },
            { dimension: '兵力规模', trend: '持续扩大', description: '从春秋数万车战，到明清数十万冷热兵器协同，战争规模扩大10倍以上' },
            { dimension: '地形偏好', trend: '多样化', description: '从早期单一平原列阵，到河谷、山地、关隘等复杂地形全面开花' },
            { dimension: '机动能力', trend: '显著增强', description: '车战→骑兵→火器，战略机动距离从百公里级扩展到千里级' },
            { dimension: '技术依赖', trend: '不断加深', description: '从人力驱动到机械蓄力再到火药化学能，军事技术对胜负影响持续放大' }
        ];

        return {
            profiles: profiles,
            change_points: changePoints,
            time_animation: timeAnim,
            time_series: timeSeries,
            summary_trends: summaryTrends
        };
    }

    function loadEvolution(callback) {
        if (state.data) {
            if (callback) callback(state.data);
            return;
        }
        fetchWithFallback('/api/doctrine_evolution', function () {
            return mockEvolutionData();
        }, function (data) {
            state.data = data;
            state.currentFrame = 0;
            if (callback) callback(data);
        });
    }

    function clearEvolutionLayers(map) {
        for (var i = 0; i < state.evolutionLayers.length; i++) {
            map.removeLayer(state.evolutionLayers[i]);
        }
        state.evolutionLayers = [];
        for (var j = 0; j < state.hotspotMarkers.length; j++) {
            map.removeLayer(state.hotspotMarkers[j]);
        }
        state.hotspotMarkers = [];
    }

    function renderFrame(map, frame) {
        clearEvolutionLayers(map);
        if (!frame) return;

        for (var i = 0; i < frame.heat_samples.length; i++) {
            var hs = frame.heat_samples[i];
            var intensity = hs.weight || 0.5;
            var r = 4 + Math.floor(intensity * 12);
            var alpha = 0.12 + intensity * 0.25;
            var m = L.circleMarker([hs.lat, hs.lng], {
                radius: r,
                fillColor: frame.era_color || '#d4af37',
                color: 'transparent',
                weight: 0,
                fillOpacity: alpha
            });
            m.addTo(map);
            state.evolutionLayers.push(m);
        }

        for (var h = 0; h < frame.hotspots.length; h++) {
            var hp = frame.hotspots[h];
            var mr = L.circleMarker([hp.lat, hp.lng], {
                radius: 16 * hp.intensity,
                fillColor: frame.era_color || '#e74c3c',
                color: '#f4e5b0',
                weight: 2,
                fillOpacity: 0.7
            }).bindTooltip(
                '<b>军事热点</b><br/>' +
                '年代: ' + frame.era + '<br/>' +
                '年份: ' + formatYear(frame.year) + '<br/>' +
                '强度: ' + (hp.intensity * 100).toFixed(0) + '%<br/>' +
                '军事思想: ' + frame.doctrine
            );
            mr.addTo(map);
            state.hotspotMarkers.push(mr);
        }
    }

    function renderEraProfiles(profiles) {
        var info = document.getElementById('doctrine-info');
        if (!info) return;

        var profilesHtml = profiles.map(function (p) {
            var eraInfo = ERA_LIST.find(function (e) { return e.era === p.era; }) || { color: '#888' };
            return '<div class="structure-card" style="border-left:4px solid ' + eraInfo.color + '">' +
                '<div style="display:flex;justify-content:space-between;align-items:center">' +
                '<span style="font-weight:bold;color:' + eraInfo.color + '">' + p.era + '</span>' +
                '<span class="doctrine-tag" style="background:' + eraInfo.color + '">' + p.doctrine_tag + '</span>' +
                '</div>' +
                '<div class="feature-grid">' +
                '<span>战役数: ' + p.battle_count + '</span>' +
                '<span>平均海拔: ' + p.avg_elevation.toFixed(0) + 'm</span>' +
                '<span>平均兵力: ' + (p.avg_troops / 10000).toFixed(1) + '万</span>' +
                '</div>' +
                '<div style="font-size:11px;color:#8b8b7a;margin-top:4px">' + p.characteristic + '</div>' +
                '</div>';
        }).join('');

        var cp = state.data.change_points || [];
        var cpHtml = cp.map(function (c) {
            return '<div class="change-point-item">' +
                '<div style="display:flex;justify-content:space-between">' +
                '<span style="font-weight:bold;color:#d4af37">' + formatYear(c.year) + '</span>' +
                '<span style="color:#8b8b7a">置信度 ' + (c.confidence * 100).toFixed(0) + '%</span>' +
                '</div>' +
                '<div style="font-size:12px;margin:3px 0">' +
                '<span style="color:#e74c3c">' + c.before_doctrine + '</span>' +
                ' → ' +
                '<span style="color:#27ae60">' + c.after_doctrine + '</span>' +
                '</div>' +
                '<div style="font-size:11px;color:#8b8b7a">' + c.era_boundary +
                ' | 突变幅度 ' + (c.change_magnitude * 100).toFixed(0) + '%' +
                '</div>' +
                '</div>';
        }).join('');

        var trends = state.data.summary_trends || [];
        var trendsHtml = trends.map(function (t) {
            return '<div class="trend-item">' +
                '<div style="font-weight:bold">' + t.dimension + ': ' +
                '<span style="color:#27ae60">' + t.trend + '</span></div>' +
                '<div style="font-size:11px;color:#8b8b7a;margin-top:2px">' + t.description + '</div>' +
                '</div>';
        }).join('');

        info.innerHTML =
            '<h4 style="color:#d4af37">朝代军事思想画像</h4>' +
            profilesHtml +
            '<h4 style="margin-top:10px;color:#d4af37">演变变点检测</h4>' +
            cpHtml +
            '<h4 style="margin-top:10px;color:#d4af37">演变趋势总结</h4>' +
            trendsHtml;
    }

    function drawTimeSeries(canvasId) {
        var canvas = document.getElementById(canvasId);
        if (!canvas || !state.data) return;
        var ctx = canvas.getContext('2d');
        var w = canvas.width, h = canvas.height;
        ctx.clearRect(0, 0, w, h);
        var ts = state.data.time_series;
        if (!ts || !ts.length) return;

        var pad = { l: 35, r: 10, t: 10, b: 25 };
        var gw = w - pad.l - pad.r;
        var gh = h - pad.t - pad.b;
        var years = ts.map(function (d) { return d.year; });
        var elevs = ts.map(function (d) { return d.avg_elevation; });
        var minY = Math.min.apply(null, years), maxY = Math.max.apply(null, years);
        var minE = Math.min.apply(null, elevs), maxE = Math.max.apply(null, elevs);

        ctx.strokeStyle = '#3a4050';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.moveTo(pad.l, pad.t);
        ctx.lineTo(pad.l, pad.t + gh);
        ctx.lineTo(pad.l + gw, pad.t + gh);
        ctx.stroke();

        ctx.fillStyle = '#8b8b7a';
        ctx.font = '10px sans-serif';
        var labelYears = [minY, Math.round((minY + maxY) / 2), maxY];
        labelYears.forEach(function (y) {
            var x = pad.l + (y - minY) / (maxY - minY) * gw;
            ctx.fillText(formatYear(y), x - 20, pad.t + gh + 14);
        });
        var elevLabels = [Math.round(minE), Math.round(maxE)];
        elevLabels.forEach(function (e, i) {
            var y = pad.t + gh - (e - minE) / (maxE - minE) * gh;
            ctx.fillText(e + 'm', 4, y + 3);
        });

        ctx.strokeStyle = '#3498db';
        ctx.lineWidth = 2;
        ctx.beginPath();
        for (var i = 0; i < ts.length; i++) {
            var x = pad.l + (ts[i].year - minY) / (maxY - minY) * gw;
            var y = pad.t + gh - (ts[i].avg_elevation - minE) / (maxE - minE) * gh;
            if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
        }
        ctx.stroke();

        if (state.currentFrame != null && state.data.time_animation) {
            var cur = state.data.time_animation[state.currentFrame];
            if (cur) {
                var cx = pad.l + (cur.year - minY) / (maxY - minY) * gw;
                ctx.strokeStyle = '#e74c3c';
                ctx.setLineDash([4, 3]);
                ctx.beginPath();
                ctx.moveTo(cx, pad.t);
                ctx.lineTo(cx, pad.t + gh);
                ctx.stroke();
                ctx.setLineDash([]);
            }
        }
    }

    function updateUI() {
        if (!state.data || !state.data.time_animation) return;
        var frame = state.data.time_animation[state.currentFrame];
        if (!frame) return;
        var yearEl = document.getElementById('evolution-year');
        var eraEl = document.getElementById('evolution-era');
        var slider = document.getElementById('evolution-timeline');
        if (yearEl) yearEl.textContent = '年份: ' + formatYear(frame.year);
        if (eraEl) eraEl.textContent = frame.era + ' · ' + frame.doctrine;
        if (slider) {
            slider.value = state.currentFrame;
            slider.max = state.data.time_animation.length - 1;
        }
    }

    function startPlay(map) {
        if (state.playing || !state.data) return;
        state.playing = true;
        document.getElementById('btn-evolution-play').textContent = '⏸';
        var interval = 1000 / state.fps;
        state.timer = setInterval(function () {
            state.currentFrame++;
            if (state.currentFrame >= state.data.time_animation.length) {
                state.currentFrame = 0;
            }
            renderFrame(map, state.data.time_animation[state.currentFrame]);
            updateUI();
            drawTimeSeries('chart-timeseries');
        }, interval);
    }

    function stopPlay(map) {
        state.playing = false;
        document.getElementById('btn-evolution-play').textContent = '▶';
        if (state.timer) {
            clearInterval(state.timer);
            state.timer = null;
        }
    }

    function stepPrev(map) {
        if (!state.data || !state.data.time_animation) return;
        state.currentFrame = Math.max(0, state.currentFrame - 1);
        renderFrame(map, state.data.time_animation[state.currentFrame]);
        updateUI();
        drawTimeSeries('chart-timeseries');
    }

    function stepNext(map) {
        if (!state.data || !state.data.time_animation) return;
        state.currentFrame = Math.min(state.data.time_animation.length - 1, state.currentFrame + 1);
        renderFrame(map, state.data.time_animation[state.currentFrame]);
        updateUI();
        drawTimeSeries('chart-timeseries');
    }

    function initUI(map, callback) {
        loadEvolution(function (data) {
            document.getElementById('evolution-controls').classList.remove('hidden');
            state.currentFrame = 0;
            renderEraProfiles(data.profiles);
            renderFrame(map, data.time_animation[0]);
            updateUI();
            drawTimeSeries('chart-timeseries');
            if (callback) callback(data);
        });

        document.getElementById('btn-evolution-play').onclick = function () {
            if (state.playing) stopPlay(map); else startPlay(map);
        };
        document.getElementById('btn-evolution-prev').onclick = function () { stepPrev(map); };
        document.getElementById('btn-evolution-next').onclick = function () { stepNext(map); };
        document.getElementById('evolution-timeline').addEventListener('input', function (e) {
            if (!state.data) return;
            stopPlay(map);
            state.currentFrame = parseInt(e.target.value);
            renderFrame(map, state.data.time_animation[state.currentFrame]);
            updateUI();
            drawTimeSeries('chart-timeseries');
        });
    }

    function reset(map) {
        stopPlay(map);
        clearEvolutionLayers(map);
        state.data = null;
        state.currentFrame = 0;
        document.getElementById('evolution-controls').classList.add('hidden');
    }

    function hasData() { return !!state.data; }

    return {
        initUI: initUI,
        reset: reset,
        startPlay: startPlay,
        stopPlay: stopPlay,
        stepPrev: stepPrev,
        stepNext: stepNext,
        renderFrame: renderFrame,
        clearEvolutionLayers: clearEvolutionLayers,
        drawTimeSeries: drawTimeSeries,
        hasData: hasData,
        formatYear: formatYear
    };
})();