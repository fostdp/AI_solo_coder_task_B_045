const BattlefieldMap = (function() {
    let map, canvas, ctx, canvasLayer;
    let regionLayer, highProbLayer;
    let hoveredBattlefield = null;
    let selectedBattlefield = null;
    let onSelectCallback = null;
    let onHoverCallback = null;

    const ERA_COLORS = {
        '春秋战国': '#e74c3c',
        '秦汉': '#e67e22',
        '三国两晋南北朝': '#f1c40f',
        '隋唐五代': '#2ecc71',
        '宋辽金元': '#3498db',
        '明清': '#9b59b6'
    };

    const ERA_NAMES = Object.keys(ERA_COLORS);

    function getEraColor(era) {
        return ERA_COLORS[era] || '#d4af37';
    }

    function init(mapContainerId, canvasId, onSelect, onHover) {
        onSelectCallback = onSelect;
        onHoverCallback = onHover;

        map = L.map(mapContainerId).setView([35.0, 110.0], 5);
        L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
            attribution: '&copy; 古战场 GIS',
            maxZoom: 10,
            minZoom: 4
        }).addTo(map);

        canvas = document.getElementById(canvasId);
        ctx = canvas.getContext('2d');
        resizeCanvas();
        canvasLayer = L.layerGroup().addTo(map);
        regionLayer = L.layerGroup();
        highProbLayer = L.layerGroup();

        map.on('moveend zoomend resize', resizeCanvas);
        canvas.addEventListener('mousemove', handleMouseMove);
        canvas.addEventListener('click', handleClick);
    }

    function resizeCanvas() {
        const size = map.getSize();
        canvas.width = size.x;
        canvas.height = size.y;
        canvas.style.left = '0';
        canvas.style.top = '0';
        canvas.style.width = size.x + 'px';
        canvas.style.height = size.y + 'px';
    }

    function latLngToCanvas(lng, lat) {
        const p = map.latLngToContainerPoint([lat, lng]);
        return { x: p.x, y: p.y };
    }

    function drawBattlefieldList(filtered, show) {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        if (!show) return;
        const bounds = map.getBounds();
        for (const bf of filtered) {
            if (!bounds.contains([bf.lat, bf.lng])) continue;
            drawTriangle(bf, bf.id === (hoveredBattlefield?.id || -1));
        }
    }

    function drawTriangle(bf, highlight) {
        const pt = latLngToCanvas(bf.lng, bf.lat);
        const size = AppData.getTroopSize(bf.total_troops);
        const s = highlight ? size * 1.3 : size;
        ctx.save();
        ctx.translate(pt.x, pt.y);
        ctx.beginPath();
        ctx.moveTo(0, -s);
        ctx.lineTo(s * 0.87, s * 0.5);
        ctx.lineTo(-s * 0.87, s * 0.5);
        ctx.closePath();
        const color = getEraColor(bf.era);
        ctx.fillStyle = color;
        if (highlight) {
            ctx.shadowColor = color;
            ctx.shadowBlur = 15;
        }
        ctx.fill();
        ctx.strokeStyle = '#f4e5b0';
        ctx.lineWidth = highlight ? 2 : 1;
        ctx.stroke();
        ctx.restore();
    }

    function handleMouseMove(e) {
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        const filtered = AppData.getFilteredBattlefields();
        let found = null;
        for (const bf of filtered) {
            const pt = latLngToCanvas(bf.lng, bf.lat);
            const size = AppData.getTroopSize(bf.total_troops);
            const dx = mx - pt.x;
            const dy = my - pt.y;
            if (Math.abs(dx) < size * 1.2 && Math.abs(dy) < size * 1.2) {
                const d = Math.sqrt(dx*dx + dy*dy);
                if (d < size * 1.2) {
                    found = bf;
                    break;
                }
            }
        }
        if (found?.id !== hoveredBattlefield?.id) {
            hoveredBattlefield = found;
            canvas.style.cursor = found ? 'pointer' : 'default';
            if (onHoverCallback) onHoverCallback(found);
        }
    }

    function handleClick(e) {
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        const filtered = AppData.getFilteredBattlefields();
        let found = null;
        for (const bf of filtered) {
            const pt = latLngToCanvas(bf.lng, bf.lat);
            const size = AppData.getTroopSize(bf.total_troops);
            const dx = mx - pt.x;
            const dy = my - pt.y;
            if (Math.sqrt(dx*dx + dy*dy) < size * 1.2) {
                found = bf;
                break;
            }
        }
        selectedBattlefield = found;
        if (onSelectCallback) onSelectCallback(found);
    }

    function drawRoads(roads) {
        for (const r of roads) {
            const ll = r.coords.map(c => [c[1], c[0]]);
            L.polyline(ll, {
                color: 'rgba(210, 180, 140, 0.6)',
                weight: 2,
                dashArray: '6, 4'
            }).bindPopup(`古代道路: ${r.road_name}`).addTo(map);
        }
    }

    function drawRivers(rivers) {
        for (const rv of rivers) {
            const ll = rv.coords.map(c => [c[1], c[0]]);
            const w = rv.importance === 'major' ? 3 : 1.5;
            const op = rv.importance === 'major' ? 0.7 : 0.4;
            L.polyline(ll, {
                color: `rgba(70, 130, 180, ${op})`,
                weight: w
            }).bindPopup(`河流: ${rv.river_name}`).addTo(map);
        }
    }

    function drawRegionsWithUncertainty(regions, fcmResult, onRegionClick) {
        regionLayer.clearLayers();
        const colors = ['rgba(231, 76, 60, 0.2)', 'rgba(52, 152, 219, 0.2)', 'rgba(46, 204, 113, 0.2)',
                        'rgba(155, 89, 182, 0.2)', 'rgba(241, 196, 15, 0.2)', 'rgba(230, 126, 34, 0.2)',
                        'rgba(26, 188, 156, 0.2)', 'rgba(211, 84, 0, 0.2)'];
        const borderColors = ['rgba(231, 76, 60, 0.8)', 'rgba(52, 152, 219, 0.8)', 'rgba(46, 204, 113, 0.8)',
                              'rgba(155, 89, 182, 0.8)', 'rgba(241, 196, 15, 0.8)', 'rgba(230, 126, 34, 0.8)',
                              'rgba(26, 188, 156, 0.8)', 'rgba(211, 84, 0, 0.8)'];

        regions.forEach((region, idx) => {
            if (!region.coords || !region.coords[0]) return;
            const latlngs = region.coords[0].map(c => [c[1], c[0]]);
            const uncertainty = region.uncertainty || (fcmResult?.uncertainties ? fcmResult.uncertainties[idx] : 0.1);
            const baseOpacity = Math.max(0.15, 0.45 - uncertainty * 0.8);
            const dashArray = uncertainty > 0.25 ? '8, 4' : (uncertainty > 0.15 ? '4, 4' : null);

            L.polygon(latlngs, {
                color: borderColors[idx % borderColors.length],
                weight: 2,
                fillColor: colors[idx % colors.length],
                fillOpacity: baseOpacity,
                dashArray: dashArray,
                opacity: Math.max(0.4, 0.8 - uncertainty * 0.8)
            }).bindPopup(renderRegionPopup(region, uncertainty)).addTo(regionLayer);

            if (uncertainty > 0.15) {
                const center = latlngs.reduce((acc, c) => [acc[0] + c[0], acc[1] + c[1]], [0, 0]);
                center[0] /= latlngs.length;
                center[1] /= latlngs.length;
                const ringRadius = 12000 + uncertainty * 40000;
                L.circle(center, {
                    radius: ringRadius,
                    color: 'rgba(231, 76, 60, ' + (0.2 + uncertainty * 0.5) + ')',
                    weight: 1,
                    fillColor: 'rgba(231, 76, 60, ' + (0.05 + uncertainty * 0.15) + ')',
                    fillOpacity: 0.3,
                    interactive: false,
                    className: 'uncertainty-layer'
                }).addTo(regionLayer);
            }

            const center = latlngs.reduce((acc, c) => [acc[0] + c[0], acc[1] + c[1]], [0, 0]);
            center[0] /= latlngs.length;
            center[1] /= latlngs.length;
            L.marker(center, {
                icon: L.divIcon({
                    html: `<div style="background:rgba(0,0,0,0.7);color:#f4e5b0;padding:2px 6px;border-radius:3px;font-size:10px;border:1px solid ${borderColors[idx % borderColors.length]};white-space:nowrap">${region.region_name}</div>`,
                    className: '',
                    iconSize: [80, 18]
                })
            }).addTo(regionLayer);
        });
        if (!map.hasLayer(regionLayer)) {
            regionLayer.addTo(map);
        }
    }

    function renderRegionPopup(region, uncertainty) {
        const badgeClass = uncertainty < 0.15 ? 'uncertainty-low' : (uncertainty < 0.25 ? 'uncertainty-medium' : 'uncertainty-high');
        return `
            <strong>${region.region_name}</strong><br>
            编码: ${region.region_code}<br>
            战役数量: ${region.battle_count}<br>
            战场密度: ${region.avg_density.toFixed(2)}<br>
            主导地形: ${region.dominant_terrain}<br>
            <div style="margin-top:4px;padding-top:4px;border-top:1px solid #333">
                聚类不确定性: <span class="uncertainty-badge ${badgeClass}">${(uncertainty * 100).toFixed(1)}%</span><br>
                平均隶属度: ${((region.avg_membership || 0.8) * 100).toFixed(1)}%
            </div>
        `;
    }

    function drawHighProbAreas(areas) {
        highProbLayer.clearLayers();
        const getColor = (p) => {
            if (p >= 0.8) return 'rgba(231, 76, 60, 0.6)';
            if (p >= 0.7) return 'rgba(241, 196, 15, 0.6)';
            return 'rgba(46, 204, 113, 0.6)';
        };
        for (const a of areas) {
            const latlngs = a.coords[0].map(c => [c[1], c[0]]);
            L.polygon(latlngs, {
                color: getColor(a.probability),
                fillColor: getColor(a.probability),
                fillOpacity: 0.35,
                weight: 1
            }).bindPopup(`${a.area_name}<br>概率: ${(a.probability*100).toFixed(1)}%`).addTo(highProbLayer);
        }
        if (!map.hasLayer(highProbLayer)) {
            highProbLayer.addTo(map);
        }
    }

    function toggleRegions(show) {
        if (show) regionLayer.addTo(map); else map.removeLayer(regionLayer);
    }

    function toggleHighProb(show) {
        if (show) highProbLayer.addTo(map); else map.removeLayer(highProbLayer);
    }

    function clearRegions() { regionLayer.clearLayers(); }
    function clearHighProb() { highProbLayer.clearLayers(); }

    function getMap() { return map; }
    function getCanvasCtx() { return ctx; }
    function getEraColors() { return ERA_COLORS; }
    function getEraNames() { return ERA_NAMES; }
    function getSelected() { return selectedBattlefield; }
    function setSelected(bf) { selectedBattlefield = bf; }
    function setHovered(bf) { hoveredBattlefield = bf; }
    function getHovered() { return hoveredBattlefield; }

    function render() {
        requestAnimationFrame(() => {
            const filtered = AppData.getFilteredBattlefields();
            const state = AppData.getState();
            drawBattlefieldList(filtered, state.showBattlefields);
        });
    }

    return {
        init, drawRoads, drawRivers,
        drawRegionsWithUncertainty, drawHighProbAreas,
        toggleRegions, toggleHighProb,
        clearRegions, clearHighProb,
        render, resizeCanvas,
        getEraColor, getMap, getCanvasCtx,
        getEraColors, getEraNames,
        getSelected, setSelected,
        getHovered, setHovered, latLngToCanvas
    };
})();
