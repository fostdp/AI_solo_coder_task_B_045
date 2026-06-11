var SupplyVisualization = (function () {
    var state = {
        data: null,
        battlefieldId: null,
        activeBattlefield: null,
        supplyLayers: [],
        visible: false
    };

    function fetchWithFallback(url, fallbackFn, callback) {
        fetch(url).then(function (r) {
            if (!r.ok) throw new Error('fail');
            return r.json();
        }).then(callback).catch(function () {
            if (fallbackFn) callback(fallbackFn());
        });
    }

    function mockSupplyData(bf) {
        var nodeTypes = ['粮仓', '武库', '兵站', '渡口'];
        function genNodes(belligerent, baseLng, baseLat, offset) {
            var nodes = [];
            for (var i = 0; i < 4; i++) {
                var ang = (i / 4) * Math.PI * 2 + offset;
                var dist = 0.3 + Math.random() * 0.4;
                nodes.push({
                    id: (belligerent === bf.belligerent_a ? 100 : 200) + i,
                    node_name: belligerent + (i === 0 ? '总粮仓' : i === 1 ? '军械库' : i === 2 ? '兵站' : '渡口'),
                    node_type: nodeTypes[i],
                    belligerent: belligerent,
                    lng: baseLng + Math.cos(ang) * dist,
                    lat: baseLat + Math.sin(ang) * dist,
                    capacity: 5000 + Math.floor(Math.random() * 20000),
                    is_bottleneck: Math.random() < 0.3
                });
            }
            return nodes;
        }

        function genRoutes(nodes, belligerent, bfLng, bfLat) {
            var routes = [];
            for (var i = 0; i < nodes.length; i++) {
                var node = nodes[i];
                var coords = [];
                var steps = 5;
                for (var s = 0; s <= steps; s++) {
                    var t = s / steps;
                    var nx = node.lng + (bfLng - node.lng) * t + Math.sin(t * Math.PI) * 0.05 * (Math.random() - 0.5);
                    var ny = node.lat + (bfLat - node.lat) * t + Math.cos(t * Math.PI) * 0.05 * (Math.random() - 0.5);
                    coords.push([nx, ny]);
                }
                routes.push({
                    id: (belligerent === bf.belligerent_a ? 10 : 20) + i,
                    route_name: node.node_name + '→战场补给线',
                    belligerent: belligerent,
                    coords: coords,
                    total_length_km: 30 + Math.random() * 80,
                    capacity: node.capacity,
                    est_time_days: 1 + Math.random() * 4,
                    efficiency: 0.5 + Math.random() * 0.4,
                    bottleneck_ids: node.is_bottleneck ? [node.id] : []
                });
            }
            return routes;
        }

        var nodesA = genNodes(bf.belligerent_a, bf.lng - 0.5, bf.lat, 0);
        var nodesB = genNodes(bf.belligerent_b, bf.lng + 0.5, bf.lat, Math.PI);
        var routesA = genRoutes(nodesA, bf.belligerent_a, bf.lng, bf.lat);
        var routesB = genRoutes(nodesB, bf.belligerent_b, bf.lng, bf.lat);

        var scoreA = routesA.reduce(function (s, r) { return s + r.efficiency; }, 0) / routesA.length;
        var scoreB = routesB.reduce(function (s, r) { return s + r.efficiency; }, 0) / routesB.length;

        return {
            battlefield_id: bf.id,
            routes_a: routesA,
            routes_b: routesB,
            nodes_a: nodesA,
            nodes_b: nodesB,
            bottlenecks_a: nodesA.filter(function (n) { return n.is_bottleneck; }),
            bottlenecks_b: nodesB.filter(function (n) { return n.is_bottleneck; }),
            advantage_side: scoreA >= scoreB ? bf.belligerent_a : bf.belligerent_b,
            advantage_score: Math.abs(scoreA - scoreB)
        };
    }

    function loadSupply(bfId, bf, callback) {
        if (state.battlefieldId === bfId && state.data) {
            if (callback) callback(state.data);
            return;
        }
        state.battlefieldId = bfId;
        state.activeBattlefield = bf;
        fetchWithFallback('/api/supply_analysis/' + bfId, function () {
            return mockSupplyData(bf);
        }, function (data) {
            state.data = data;
            if (callback) callback(data);
        });
    }

    function clearSupplyLayers(map) {
        for (var i = 0; i < state.supplyLayers.length; i++) {
            map.removeLayer(state.supplyLayers[i]);
        }
        state.supplyLayers = [];
    }

    function renderSupply(map, show) {
        clearSupplyLayers(map);
        state.visible = show;
        if (!show || !state.data) return;

        var colorA = '#3498db';
        var colorB = '#e74c3c';

        function drawRoutes(routes, color) {
            for (var i = 0; i < routes.length; i++) {
                var r = routes[i];
                var latlngs = r.coords.map(function (c) { return [c[1], c[0]]; });
                var hasBn = r.bottleneck_ids && r.bottleneck_ids.length > 0;
                var pline = L.polyline(latlngs, {
                    color: color,
                    weight: hasBn ? 5 : 3,
                    opacity: hasBn ? 0.95 : 0.7,
                    dashArray: hasBn ? null : '10, 6'
                }).bindTooltip('<b>' + r.route_name + '</b><br/>长度: ' + r.total_length_km.toFixed(1) + ' km<br/>时效: ' + r.est_time_days.toFixed(1) + ' 天<br/>效率: ' + (r.efficiency * 100).toFixed(0) + '%' + (hasBn ? '<br/><span style="color:#e74c3c">⚠ 含瓶颈节点</span>' : ''));
                pline.addTo(map);
                state.supplyLayers.push(pline);
            }
        }

        function drawNodes(nodes, color) {
            for (var i = 0; i < nodes.length; i++) {
                var n = nodes[i];
                var isBn = n.is_bottleneck;
                var radius = isBn ? 12 : 8;
                var marker = L.circleMarker([n.lat, n.lng], {
                    radius: radius,
                    fillColor: isBn ? '#e67e22' : color,
                    color: isBn ? '#e74c3c' : '#f4e5b0',
                    weight: isBn ? 3 : 2,
                    fillOpacity: 0.9
                }).bindTooltip('<b>' + n.node_name + '</b><br/>类型: ' + n.node_type + '<br/>容量: ' + n.capacity.toLocaleString() + ' 石/人' + (isBn ? '<br/><span style="color:#e74c3c">⚠ 瓶颈节点</span>' : ''));
                marker.addTo(map);
                state.supplyLayers.push(marker);
            }
        }

        drawRoutes(state.data.routes_a, colorA);
        drawRoutes(state.data.routes_b, colorB);
        drawNodes(state.data.nodes_a, colorA);
        drawNodes(state.data.nodes_b, colorB);
    }

    function renderSupplyPanel() {
        if (!state.data) return;
        var info = document.getElementById('supply-info');
        if (!info) return;
        var d = state.data;
        var bf = state.activeBattlefield;

        function nodesHtml(nodes, bnList, color, side) {
            return nodes.map(function (n) {
                var isBn = n.is_bottleneck;
                return '<div class="structure-card" style="border-left:4px solid ' + (isBn ? '#e67e22' : color) + '">' +
                    '<div style="font-weight:bold">' + n.node_name + '</div>' +
                    '<div style="font-size:11px;color:#8b8b7a">类型: ' + n.node_type + ' | 容量: ' + n.capacity.toLocaleString() + '</div>' +
                    (isBn ? '<div style="color:#e74c3c;font-size:11px">⚠ 瓶颈节点</div>' : '') +
                    '</div>';
            }).join('');
        }

        info.innerHTML =
            '<div class="advantage-badge ' + (d.advantage_side === bf.belligerent_a ? 'a' : 'b') + '">' +
            '补给优势方: ' + d.advantage_side + ' (优势 ' + (d.advantage_score * 100).toFixed(1) + '%)' +
            '</div>' +
            '<div class="score-bar"><div class="score-fill" style="width:50%;background:#3498db"></div><div class="score-fill" style="width:' + (50 + d.advantage_score * 50 * (d.advantage_side === bf.belligerent_a ? 1 : -1)) + '%;background:' + (d.advantage_side === bf.belligerent_a ? '#3498db' : '#e74c3c') + ';right:0"></div></div>' +
            '<h4 style="margin-top:10px;color:#3498db">' + bf.belligerent_a + ' 补给体系</h4>' +
            nodesHtml(d.nodes_a, d.bottlenecks_a, '#3498db', 'a') +
            '<h4 style="margin-top:10px;color:#e74c3c">' + bf.belligerent_b + ' 补给体系</h4>' +
            nodesHtml(d.nodes_b, d.bottlenecks_b, '#e74c3c', 'b');
        info.classList.remove('hidden');

        var panelInfo = document.getElementById('panel-supply-info');
        if (panelInfo) {
            panelInfo.innerHTML = info.innerHTML;
        }
    }

    function initUI(map, bf, callback) {
        loadSupply(bf.id, bf, function (data) {
            renderSupply(map, true);
            renderSupplyPanel();
            document.getElementById('toggle-supply').checked = true;
            if (callback) callback(data);
        });
    }

    function toggle(map, show) {
        renderSupply(map, show);
        if (show && state.data) {
            renderSupplyPanel();
        }
    }

    function reset(map) {
        clearSupplyLayers(map);
        state.data = null;
        state.battlefieldId = null;
        state.activeBattlefield = null;
        state.visible = false;
        var info = document.getElementById('supply-info');
        if (info) info.classList.add('hidden');
    }

    function isVisible() { return state.visible; }
    function hasData() { return !!state.data; }

    return {
        initUI: initUI,
        reset: reset,
        toggle: toggle,
        renderSupply: renderSupply,
        clearSupplyLayers: clearSupplyLayers,
        isVisible: isVisible,
        hasData: hasData
    };
})();
