var DefenseVisualization = (function () {
    var state = {
        data: null,
        structures: null,
        battlefieldId: null,
        activeBattlefield: null,
        defenseLayers: [],
        blindzoneLayers: [],
        structuresVisible: false,
        blindzonesVisible: false
    };

    function fetchWithFallback(url, fallbackFn, callback) {
        fetch(url).then(function (r) {
            if (!r.ok) throw new Error('fail');
            return r.json();
        }).then(callback).catch(function () {
            if (fallbackFn) callback(fallbackFn());
        });
    }

    function mockStructures(bf) {
        var types = ['城墙', '关隘', '堡垒', '烽火台', '要塞', '寨堡', '护城河'];
        var materials = ['夯土', '砖石', '土石混合', '条石', '包砖'];
        var arr = [];
        var count = 3 + Math.floor(Math.random() * 3);
        for (var i = 0; i < count; i++) {
            var ang = (i / count) * Math.PI * 2;
            var dist = 0.08 + Math.random() * 0.15;
            var lng = bf.lng + Math.cos(ang) * dist;
            var lat = bf.lat + Math.sin(ang) * dist;
            var type = types[i % types.length];
            var isPoint = type === '烽火台' || type === '堡垒';
            var coords = null;
            if (!isPoint) {
                coords = [];
                var r = 0.02 + Math.random() * 0.03;
                for (var v = 0; v < 16; v++) {
                    var theta = v * Math.PI * 2 / 16;
                    var rx = r * (0.8 + Math.random() * 0.4);
                    var ry = r * (0.6 + Math.random() * 0.4);
                    coords.push([lng + rx * Math.cos(theta), lat + ry * Math.sin(theta)]);
                }
            }
            arr.push({
                id: 100 + i,
                battlefield_id: bf.id,
                structure_type: type,
                structure_name: type + '-' + (i + 1) + '号',
                height_m: 3 + Math.random() * 12,
                length_m: isPoint ? 0 : 200 + Math.random() * 1500,
                thickness_m: 1 + Math.random() * 8,
                material: materials[Math.floor(Math.random() * materials.length)],
                gate_count: isPoint ? 0 : Math.floor(Math.random() * 4),
                tower_count: isPoint ? 1 : Math.floor(Math.random() * 8),
                lng: lng,
                lat: lat,
                coords: coords
            });
        }
        return arr;
    }

    function mockDefenseEvaluation(bf, structures) {
        var blindZones = [];
        for (var si = 0; si < Math.min(2, structures.length); si++) {
            var s = structures[si];
            var numBz = 1 + Math.floor(Math.random() * 2);
            for (var bi = 0; bi < numBz; bi++) {
                var startDir = Math.random() * 360;
                var spread = 30 + Math.random() * 60;
                var maxDist = 2 + Math.random() * 4;
                var visPct = 0.1 + Math.random() * 0.35;
                var risk = visPct < 0.2 ? '极高' : visPct < 0.3 ? '高' : visPct < 0.4 ? '中' : '低';
                var coords = [];
                var steps = 20;
                coords.push([s.lng, s.lat]);
                for (var st = 0; st <= steps; st++) {
                    var theta = (startDir + (st / steps) * spread) * Math.PI / 180;
                    coords.push([
                        s.lng + maxDist * Math.cos(theta) / 90,
                        s.lat + maxDist * Math.sin(theta) / 111
                    ]);
                }
                coords.push([s.lng, s.lat]);
                blindZones.push({
                    id: 1000 + si * 10 + bi,
                    structure_id: s.id,
                    structure_name: s.structure_name,
                    direction: startDir.toFixed(0) + '°-' + (startDir + spread).toFixed(0) + '°',
                    area_km2: Math.PI * maxDist * maxDist * spread / 360,
                    max_distance_km: maxDist,
                    visibility_pct: visPct,
                    risk_level: risk,
                    coords: coords
                });
            }
        }

        var overall = 0.5 + Math.random() * 0.35;
        var visScore = 0.4 + Math.random() * 0.45;
        var structScore = 0.55 + Math.random() * 0.35;
        var topoScore = 0.45 + Math.random() * 0.4;

        var recs = [];
        if (visScore < 0.6) recs.push('增设瞭望塔，扩展观察视野覆盖盲区');
        if (structScore < 0.65) recs.push('加固城墙厚度与高度，使用砖石材料');
        if (topoScore < 0.6) recs.push('利用地形高差，增设外围防御工事');
        if (blindZones.some(function (b) { return b.risk_level === '极高' || b.risk_level === '高'; })) recs.push('在高风险盲区增设侧防堡垒与交叉火力点');
        if (recs.length === 0) recs.push('现有防御体系完善，建议定期维护');

        return {
            battlefield_id: bf.id,
            overall_score: overall,
            visibility_score: visScore,
            structural_score: structScore,
            topographic_score: topoScore,
            blind_zones: blindZones,
            recommendations: recs
        };
    }

    function loadDefense(bfId, bf, callback) {
        if (state.battlefieldId === bfId && state.data && state.structures) {
            if (callback) callback(state.data, state.structures);
            return;
        }
        state.battlefieldId = bfId;
        state.activeBattlefield = bf;
        var pending = 2;
        var result = { eval: null, struct: null };

        function done() {
            pending--;
            if (pending === 0 && callback) callback(result.eval, result.struct);
        }

        fetchWithFallback('/api/defense_evaluation/' + bfId, function () {
            return mockDefenseEvaluation(bf, state.structures || mockStructures(bf));
        }, function (d) {
            state.data = d;
            result.eval = d;
            done();
        });

        fetchWithFallback('/api/military_structures/' + bfId, function () {
            return mockStructures(bf);
        }, function (d) {
            state.structures = d;
            result.struct = d;
            done();
        });
    }

    function clearDefenseLayers(map) {
        for (var i = 0; i < state.defenseLayers.length; i++) map.removeLayer(state.defenseLayers[i]);
        state.defenseLayers = [];
    }

    function clearBlindzoneLayers(map) {
        for (var i = 0; i < state.blindzoneLayers.length; i++) map.removeLayer(state.blindzoneLayers[i]);
        state.blindzoneLayers = [];
    }

    function renderStructures(map, show) {
        clearDefenseLayers(map);
        state.structuresVisible = show;
        if (!show || !state.structures) return;

        for (var i = 0; i < state.structures.length; i++) {
            var s = state.structures[i];
            var isPoint = !s.coords;
            if (isPoint) {
                var color = '#8b4513';
                if (s.structure_type === '烽火台') color = '#d35400';
                if (s.structure_type === '堡垒') color = '#7f8c8d';
                var m = L.circleMarker([s.lat, s.lng], {
                    radius: 9,
                    fillColor: color,
                    color: '#f4e5b0',
                    weight: 2,
                    fillOpacity: 0.9
                }).bindTooltip(
                    '<b>' + s.structure_name + '</b><br/>' +
                    '类型: ' + s.structure_type + '<br/>' +
                    '材质: ' + s.material + '<br/>' +
                    '高度: ' + s.height_m.toFixed(1) + ' m'
                );
                m.addTo(map);
                state.defenseLayers.push(m);
            } else {
                var latlngs = s.coords.map(function (c) { return [c[1], c[0]]; });
                var fillColor = s.structure_type === '城墙' ? '#8b4513' :
                    s.structure_type === '关隘' ? '#654321' :
                    s.structure_type === '护城河' ? '#4682b4' : '#556b2f';
                var poly = L.polygon(latlngs, {
                    color: '#d4af37',
                    weight: 2,
                    fillColor: fillColor,
                    fillOpacity: 0.55
                }).bindTooltip(
                    '<b>' + s.structure_name + '</b><br/>' +
                    '类型: ' + s.structure_type + '<br/>' +
                    '材质: ' + s.material + '<br/>' +
                    '高度: ' + s.height_m.toFixed(1) + ' m<br/>' +
                    '长度: ' + s.length_m.toFixed(0) + ' m<br/>' +
                    '厚度: ' + s.thickness_m.toFixed(1) + ' m<br/>' +
                    '城门: ' + s.gate_count + ' | 敌楼: ' + s.tower_count
                );
                poly.addTo(map);
                state.defenseLayers.push(poly);
            }
        }
    }

    function renderBlindZones(map, show) {
        clearBlindzoneLayers(map);
        state.blindzonesVisible = show;
        if (!show || !state.data) return;

        var riskColors = {
            '低': 'rgba(46, 204, 113, 0.35)',
            '中': 'rgba(241, 196, 15, 0.45)',
            '高': 'rgba(230, 126, 34, 0.55)',
            '极高': 'rgba(231, 76, 60, 0.65)'
        };
        var riskBorders = {
            '低': '#27ae60',
            '中': '#f1c40f',
            '高': '#e67e22',
            '极高': '#e74c3c'
        };

        for (var i = 0; i < state.data.blind_zones.length; i++) {
            var bz = state.data.blind_zones[i];
            var latlngs = bz.coords.map(function (c) { return [c[1], c[0]]; });
            var poly = L.polygon(latlngs, {
                color: riskBorders[bz.risk_level] || '#999',
                weight: 2,
                dashArray: '6, 4',
                fillColor: riskColors[bz.risk_level] || 'rgba(150,150,150,0.4)',
                fillOpacity: 0.5
            }).bindTooltip(
                '<b>防御盲区</b><br/>' +
                '所属工程: ' + bz.structure_name + '<br/>' +
                '方向: ' + bz.direction + '<br/>' +
                '面积: ' + bz.area_km2.toFixed(2) + ' km²<br/>' +
                '最大距离: ' + bz.max_distance_km.toFixed(1) + ' km<br/>' +
                '通视率: ' + (bz.visibility_pct * 100).toFixed(0) + '%<br/>' +
                '风险等级: <span style="color:' + riskBorders[bz.risk_level] + '">' + bz.risk_level + '</span>'
            );
            poly.addTo(map);
            state.blindzoneLayers.push(poly);
        }
    }

    function renderDefensePanel() {
        if (!state.data || !state.structures) return;
        var info = document.getElementById('defense-info');
        if (!info) return;
        var d = state.data;

        function scoreBar(label, score, color) {
            return '<div style="margin-bottom:8px">' +
                '<div style="display:flex;justify-content:space-between;font-size:12px"><span>' + label + '</span><span>' + (score * 100).toFixed(0) + '%</span></div>' +
                '<div style="background:#1a1e27;border-radius:3px;height:8px;overflow:hidden;margin-top:2px">' +
                '<div style="height:100%;width:' + (score * 100) + '%;background:' + color + '"></div>' +
                '</div></div>';
        }

        var structuresHtml = state.structures.map(function (s) {
            return '<div class="structure-card">' +
                '<div style="font-weight:bold;color:#d4af37">' + s.structure_name + '</div>' +
                '<div class="feature-grid">' +
                '<span>类型: ' + s.structure_type + '</span>' +
                '<span>材质: ' + s.material + '</span>' +
                '<span>高: ' + s.height_m.toFixed(0) + 'm</span>' +
                (s.length_m > 0 ? '<span>长: ' + s.length_m.toFixed(0) + 'm</span>' : '') +
                '<span>厚: ' + s.thickness_m.toFixed(1) + 'm</span>' +
                (s.gate_count > 0 ? '<span>门: ' + s.gate_count + '</span>' : '') +
                (s.tower_count > 0 ? '<span>楼: ' + s.tower_count + '</span>' : '') +
                '</div></div>';
        }).join('');

        var bzHtml = d.blind_zones.length ? d.blind_zones.map(function (bz) {
            return '<div class="blindzone-item risk-' + bz.risk_level + '">' +
                '<div style="font-weight:bold">' + bz.structure_name + ' - ' + bz.direction + '</div>' +
                '<div class="feature-grid">' +
                '<span>面积: ' + bz.area_km2.toFixed(2) + 'km²</span>' +
                '<span>距: ' + bz.max_distance_km.toFixed(1) + 'km</span>' +
                '<span>通视: ' + (bz.visibility_pct * 100).toFixed(0) + '%</span>' +
                '<span>风险: ' + bz.risk_level + '</span>' +
                '</div></div>';
        }).join('') : '<div style="color:#27ae60">未检测到显著防御盲区</div>';

        var recHtml = d.recommendations.map(function (r) {
            return '<div class="trend-item">• ' + r + '</div>';
        }).join('');

        info.innerHTML =
            '<div style="text-align:center;margin-bottom:10px">' +
            '<div style="font-size:28px;font-weight:bold;color:#d4af37">' + (d.overall_score * 100).toFixed(0) + '</div>' +
            '<div style="font-size:11px;color:#8b8b7a">综合防御效能评分</div></div>' +
            scoreBar('通视覆盖', d.visibility_score, '#3498db') +
            scoreBar('结构强度', d.structural_score, '#e67e22') +
            scoreBar('地形优势', d.topographic_score, '#27ae60') +
            '<h4 style="margin-top:10px;color:#d4af37">军事工程</h4>' +
            structuresHtml +
            '<h4 style="margin-top:10px;color:#d4af37">防御盲区</h4>' +
            bzHtml +
            '<h4 style="margin-top:10px;color:#d4af37">改进建议</h4>' +
            recHtml;
        info.classList.remove('hidden');

        var pInfo = document.getElementById('panel-defense-info');
        if (pInfo) pInfo.innerHTML = info.innerHTML;
    }

    function initUI(map, bf, callback) {
        loadDefense(bf.id, bf, function (evalData, structData) {
            renderStructures(map, true);
            renderBlindZones(map, true);
            renderDefensePanel();
            document.getElementById('toggle-structures').checked = true;
            document.getElementById('toggle-blindzones').checked = true;
            if (callback) callback(evalData, structData);
        });
    }

    function toggleStructures(map, show) { renderStructures(map, show); }
    function toggleBlindZones(map, show) { renderBlindZones(map, show); }

    function reset(map) {
        clearDefenseLayers(map);
        clearBlindzoneLayers(map);
        state.data = null;
        state.structures = null;
        state.battlefieldId = null;
        state.activeBattlefield = null;
        state.structuresVisible = false;
        state.blindzonesVisible = false;
        var info = document.getElementById('defense-info');
        if (info) info.classList.add('hidden');
    }

    return {
        initUI: initUI,
        reset: reset,
        toggleStructures: toggleStructures,
        toggleBlindZones: toggleBlindZones,
        renderStructures: renderStructures,
        renderBlindZones: renderBlindZones,
        clearDefenseLayers: clearDefenseLayers,
        clearBlindzoneLayers: clearBlindzoneLayers
    };
})();
