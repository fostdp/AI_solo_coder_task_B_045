const TerrainProfile = (function() {
    let offscreenCanvas = null;
    let offscreenCtx = null;
    let lastProfileHash = '';
    let renderQueue = [];
    let renderScheduled = false;
    const DEFAULT_OPTIONS = {
        useOffscreen: true,
        width: null,
        height: null
    };

    function isOffscreenCanvasSupported() {
        return typeof OffscreenCanvas !== 'undefined';
    }

    function initOffscreen(w, h) {
        if (offscreenCanvas && offscreenCanvas.width === w && offscreenCanvas.height === h) {
            return;
        }
        if (isOffscreenCanvasSupported()) {
            offscreenCanvas = new OffscreenCanvas(w, h);
            offscreenCtx = offscreenCanvas.getContext('2d');
        } else {
            offscreenCanvas = document.createElement('canvas');
            offscreenCanvas.width = w;
            offscreenCanvas.height = h;
            offscreenCtx = offscreenCanvas.getContext('2d');
        }
        lastProfileHash = '';
    }

    function clearOffscreenCache() {
        lastProfileHash = '';
        if (offscreenCtx) {
            offscreenCtx.clearRect(0, 0, offscreenCanvas.width, offscreenCanvas.height);
        }
    }

    function hashProfile(profile) {
        if (!profile || !profile.points || profile.points.length === 0) return 'empty';
        const firstElev = profile.points[0].elevation || 0;
        const lastElev = profile.points[profile.points.length - 1].elevation || 0;
        let checksum = 0;
        const step = Math.max(1, Math.floor(profile.points.length / 10));
        for (let i = 0; i < profile.points.length; i += step) {
            checksum = (checksum * 31 + Math.round(profile.points[i].elevation)) >>> 0;
        }
        return `${profile.min_elev.toFixed(0)}_${profile.max_elev.toFixed(0)}_${profile.points.length}_${firstElev.toFixed(0)}_${lastElev.toFixed(0)}_${checksum}`;
    }

    function requestEfficientRender(fn) {
        renderQueue.push(fn);
        if (!renderScheduled) {
            renderScheduled = true;
            requestAnimationFrame(() => {
                const tasks = renderQueue;
                renderQueue = [];
                renderScheduled = false;
                for (const t of tasks) {
                    try { t(); } catch (e) { console.error(e); }
                }
            });
        }
    }

    function throttle(fn, delay) {
        let last = 0;
        let timer = null;
        return function(...args) {
            const now = Date.now();
            const remain = delay - (now - last);
            if (remain <= 0) {
                clearTimeout(timer);
                timer = null;
                last = now;
                fn.apply(this, args);
            } else if (!timer) {
                timer = setTimeout(() => {
                    last = Date.now();
                    timer = null;
                    fn.apply(this, args);
                }, remain);
            }
        };
    }

    function drawTerrainProfile(canvas, profile, userOptions) {
        const opts = Object.assign({}, DEFAULT_OPTIONS, userOptions || {});
        const w = opts.width || canvas.clientWidth || canvas.width || 320;
        const h = opts.height || canvas.clientHeight || canvas.height || 180;
        if (canvas.width !== w) canvas.width = w;
        if (canvas.height !== h) canvas.height = h;

        const ctx = canvas.getContext('2d');
        const hash = hashProfile(profile);

        if (opts.useOffscreen && isOffscreenCanvasSupported()) {
            requestEfficientRender(() => {
                _drawTerrainProfileOffscreen(canvas, profile, hash, w, h);
            });
        } else {
            requestEfficientRender(() => {
                _drawTerrainProfileDirect(ctx, w, h, profile);
            });
        }
    }

    function _drawTerrainProfileOffscreen(canvas, profile, hash, w, h) {
        initOffscreen(w, h);
        if (lastProfileHash !== hash) {
            _drawTerrainProfileToContext(offscreenCtx, w, h, profile);
            lastProfileHash = hash;
        }
        const ctx = canvas.getContext('2d');
        ctx.clearRect(0, 0, w, h);
        ctx.drawImage(offscreenCanvas, 0, 0);
    }

    function _drawTerrainProfileDirect(ctx, w, h, profile) {
        ctx.clearRect(0, 0, w, h);
        _drawTerrainProfileToContext(ctx, w, h, profile);
    }

    function _drawTerrainProfileToContext(ctx, w, h, profile) {
        ctx.clearRect(0, 0, w, h);

        const padding = { top: 28, right: 12, bottom: 28, left: 48 };
        const chartW = w - padding.left - padding.right;
        const chartH = h - padding.top - padding.bottom;

        if (!profile || !profile.points || profile.points.length < 2) {
            ctx.fillStyle = '#8b8b7a';
            ctx.font = '12px serif';
            ctx.textAlign = 'center';
            ctx.fillText('数据不足', w / 2, h / 2);
            return;
        }

        const bg = ctx.createLinearGradient(0, padding.top, 0, h - padding.bottom);
        bg.addColorStop(0, 'rgba(212, 175, 55, 0.15)');
        bg.addColorStop(1, 'rgba(74, 44, 26, 0.05)');
        ctx.fillStyle = bg;
        ctx.fillRect(padding.left, padding.top, chartW, chartH);

        ctx.strokeStyle = 'rgba(212, 175, 55, 0.15)';
        ctx.lineWidth = 0.5;
        ctx.setLineDash([2, 2]);
        for (let i = 0; i <= 4; i++) {
            const y = padding.top + (chartH * i / 4);
            ctx.beginPath();
            ctx.moveTo(padding.left, y);
            ctx.lineTo(w - padding.right, y);
            ctx.stroke();
            const v = profile.max_elev - (profile.max_elev - profile.min_elev) * i / 4;
            ctx.setLineDash([]);
            ctx.fillStyle = '#c8c8b8';
            ctx.font = '10px monospace';
            ctx.textAlign = 'right';
            ctx.fillText(`${Math.round(v)}m`, padding.left - 4, y + 3);
            ctx.setLineDash([2, 2]);
        }
        for (let i = 0; i <= 4; i++) {
            const x = padding.left + chartW * i / 4;
            ctx.beginPath();
            ctx.moveTo(x, padding.top);
            ctx.lineTo(x, h - padding.bottom);
            ctx.stroke();
            const d = profile.total_dist * i / 4;
            ctx.setLineDash([]);
            ctx.fillStyle = '#c8c8b8';
            ctx.font = '10px monospace';
            ctx.textAlign = 'center';
            ctx.fillText(`${d.toFixed(1)}km`, x, h - padding.bottom + 14);
            ctx.setLineDash([2, 2]);
        }
        ctx.setLineDash([]);

        const pts = profile.points;
        const elevRange = profile.max_elev - profile.min_elev || 1;
        const getX = (i) => padding.left + chartW * i / (pts.length - 1);
        const getY = (e) => padding.top + chartH - chartH * (e - profile.min_elev) / elevRange;

        ctx.beginPath();
        ctx.moveTo(getX(0), getY(pts[0].elevation));
        for (let i = 1; i < pts.length; i++) {
            ctx.lineTo(getX(i), getY(pts[i].elevation));
        }
        const lineGrad = ctx.createLinearGradient(0, padding.top, 0, h - padding.bottom);
        lineGrad.addColorStop(0, '#d4af37');
        lineGrad.addColorStop(1, '#8b6914');
        ctx.strokeStyle = lineGrad;
        ctx.lineWidth = 1.8;
        ctx.stroke();

        ctx.lineTo(getX(pts.length - 1), h - padding.bottom);
        ctx.lineTo(getX(0), h - padding.bottom);
        ctx.closePath();
        const fillGrad = ctx.createLinearGradient(0, padding.top, 0, h - padding.bottom);
        fillGrad.addColorStop(0, 'rgba(212, 175, 55, 0.45)');
        fillGrad.addColorStop(1, 'rgba(74, 44, 26, 0.05)');
        ctx.fillStyle = fillGrad;
        ctx.fill();

        ctx.fillStyle = '#f4e5b0';
        ctx.font = 'bold 11px monospace';
        ctx.textAlign = 'left';
        const midX = padding.left + chartW / 2;
        ctx.fillText(`最低: ${Math.round(profile.min_elev)}m`, padding.left, 14);
        ctx.textAlign = 'center';
        ctx.fillText(`最高: ${Math.round(profile.max_elev)}m`, midX, 14);
        ctx.textAlign = 'right';
        ctx.fillText(`平均: ${Math.round(profile.avg_elev)}m`, w - padding.right, 14);

        ctx.fillStyle = '#8b8b7a';
        ctx.font = '10px serif';
        ctx.textAlign = 'center';
        ctx.fillText(`距离(km)`, midX, h - 4);
    }

    function drawUncertaintyRing(canvas, centerX, centerY, radius, uncertainty) {
        const ctx = canvas.getContext('2d');
        const grad = ctx.createRadialGradient(centerX, centerY, radius * 0.3, centerX, centerY, radius);
        const alpha = Math.min(0.6, 0.1 + uncertainty * 0.6);
        grad.addColorStop(0, `rgba(231, 76, 60, 0)`);
        grad.addColorStop(0.6, `rgba(231, 76, 60, ${alpha * 0.4})`);
        grad.addColorStop(1, `rgba(231, 76, 60, ${alpha})`);
        ctx.beginPath();
        ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
        ctx.fillStyle = grad;
        ctx.fill();
        ctx.strokeStyle = `rgba(231, 76, 60, ${0.2 + uncertainty * 0.5})`;
        ctx.lineWidth = 1 + uncertainty * 1.5;
        ctx.setLineDash([6, 4]);
        ctx.stroke();
        ctx.setLineDash([]);
    }

    return {
        isOffscreenCanvasSupported,
        requestEfficientRender,
        throttle,
        hashProfile,
        drawTerrainProfile,
        drawUncertaintyRing,
        clearOffscreenCache
    };
})();
