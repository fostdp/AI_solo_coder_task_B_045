const Charts = (function() {
    function drawBarChart(canvas, data, options) {
        if (!canvas || !data || data.length === 0) return;
        const ctx = canvas.getContext('2d');
        const w = canvas.clientWidth || canvas.width;
        const h = canvas.clientHeight || canvas.height;
        canvas.width = w;
        canvas.height = h;

        let opts = options || {};
        const padding = opts.padding || { top: 12, right: 8, bottom: 36, left: 36 };
        const barColor = opts.barColor || '#d4af37';
        const labelColor = opts.labelColor || '#f4e5b0';
        const title = opts.title || '';

        ctx.clearRect(0, 0, w, h);

        if (title) {
            ctx.fillStyle = labelColor;
            ctx.font = 'bold 12px serif';
            ctx.textAlign = 'center';
            ctx.fillText(title, w / 2, 10);
            padding.top = 22;
        }

        const cw = w - padding.left - padding.right;
        const ch = h - padding.top - padding.bottom;
        const total = data.reduce((s, d) => s + d.value, 0) || 1;
        const maxV = Math.max(...data.map(d => d.value));

        ctx.strokeStyle = 'rgba(212, 175, 55, 0.15)';
        ctx.lineWidth = 0.5;
        for (let i = 0; i <= 4; i++) {
            const y = padding.top + ch * i / 4;
            ctx.beginPath();
            ctx.moveTo(padding.left, y);
            ctx.lineTo(w - padding.right, y);
            ctx.stroke();
            ctx.fillStyle = '#8b8b7a';
            ctx.font = '10px monospace';
            ctx.textAlign = 'right';
            const v = maxV * (1 - i / 4);
            ctx.fillText(Math.round(v).toString(), padding.left - 4, y + 3);
        }

        const bw = cw / data.length * 0.7;
        const gap = cw / data.length * 0.3;

        data.forEach((d, i) => {
            const x = padding.left + i * (bw + gap) + gap / 2;
            const bh = ch * d.value / maxV;
            const y = padding.top + ch - bh;

            const grad = ctx.createLinearGradient(0, y, 0, y + bh);
            grad.addColorStop(0, d.color || barColor);
            grad.addColorStop(1, 'rgba(212, 175, 55, 0.2)');
            ctx.fillStyle = grad;
            ctx.fillRect(x, y, bw, bh);
            ctx.strokeStyle = d.color || barColor;
            ctx.lineWidth = 0.8;
            ctx.strokeRect(x, y, bw, bh);

            ctx.fillStyle = '#f4e5b0';
            ctx.font = 'bold 10px monospace';
            ctx.textAlign = 'center';
            ctx.fillText(d.value.toString(), x + bw / 2, y - 3);

            ctx.fillStyle = '#c8c8b8';
            ctx.font = '10px serif';
            ctx.save();
            ctx.translate(x + bw / 2, h - padding.bottom + 10);
            ctx.rotate(-Math.PI / 6);
            ctx.textAlign = 'right';
            ctx.fillText(d.label, 0, 0);
            ctx.restore();
        });
    }

    function drawPieChart(canvas, data, options) {
        if (!canvas || !data || data.length === 0) return;
        const ctx = canvas.getContext('2d');
        const w = canvas.clientWidth || canvas.width;
        const h = canvas.clientHeight || canvas.height;
        canvas.width = w;
        canvas.height = h;

        let opts = options || {};
        const title = opts.title || '';
        const cx = w / 2;
        const cy = h / 2 + (title ? 8 : 0);
        const r = Math.min(cx, cy - (title ? 20 : 8)) - 4;

        ctx.clearRect(0, 0, w, h);

        if (title) {
            ctx.fillStyle = '#f4e5b0';
            ctx.font = 'bold 12px serif';
            ctx.textAlign = 'center';
            ctx.fillText(title, w / 2, 12);
        }

        const total = data.reduce((s, d) => s + d.value, 0) || 1;
        let start = -Math.PI / 2;

        data.forEach((d) => {
            const angle = (d.value / total) * Math.PI * 2;
            ctx.beginPath();
            ctx.moveTo(cx, cy);
            ctx.arc(cx, cy, r, start, start + angle);
            ctx.closePath();
            ctx.fillStyle = d.color || '#d4af37';
            ctx.fill();
            ctx.strokeStyle = '#1a1e27';
            ctx.lineWidth = 1;
            ctx.stroke();

            const midA = start + angle / 2;
            const lx = cx + Math.cos(midA) * (r * 0.6);
            const ly = cy + Math.sin(midA) * (r * 0.6);
            if (angle > 0.25) {
                ctx.fillStyle = '#fff';
                ctx.font = 'bold 10px monospace';
                ctx.textAlign = 'center';
                ctx.fillText(`${((d.value / total) * 100).toFixed(0)}%`, lx, ly + 3);
            }
            start += angle;
        });

        ctx.beginPath();
        ctx.arc(cx, cy, r * 0.35, 0, Math.PI * 2);
        ctx.fillStyle = '#1a1e27';
        ctx.fill();
        ctx.fillStyle = '#f4e5b0';
        ctx.font = 'bold 11px serif';
        ctx.textAlign = 'center';
        ctx.fillText(total.toString(), cx, cy + 4);
    }

    function drawFactorContribution(canvas, factors, options) {
        if (!canvas || !factors) return;
        const data = factors.map(f => ({
            label: f.factor_name,
            value: Math.round(f.contribution * 100),
            color: getFactorColor(f)
        }));
        drawBarChart(canvas, data, Object.assign({ title: '选址因素贡献度(%)' }, options || {}));
    }

    function getFactorColor(f) {
        if (f.stability_score !== undefined) {
            if (f.stability_score >= 0.85) return '#27ae60';
            if (f.stability_score >= 0.7) return '#f1c40f';
            return '#e74c3c';
        }
        return '#d4af37';
    }

    function drawEraDistribution(canvas, battlefields) {
        const eras = {};
        for (const bf of battlefields) {
            eras[bf.era] = (eras[bf.era] || 0) + 1;
        }
        const eraOrder = ['春秋战国', '秦汉', '三国两晋南北朝', '隋唐五代', '宋辽金元', '明清'];
        const colors = ['#e74c3c', '#e67e22', '#f1c40f', '#2ecc71', '#3498db', '#9b59b6'];
        const data = eraOrder.map((e, i) => ({
            label: e, value: eras[e] || 0, color: colors[i]
        }));
        drawBarChart(canvas, data, { title: '按年代分布' });
    }

    function drawTerrainDistribution(canvas, battlefields) {
        const terr = {};
        for (const bf of battlefields) {
            terr[bf.terrain_type] = (terr[bf.terrain_type] || 0) + 1;
        }
        const names = ['山地', '平原', '河谷', '关隘'];
        const colors = ['#8b4513', '#c8b07a', '#4682b4', '#6b3410'];
        const data = names.map((n, i) => ({
            label: n, value: terr[n] || 0, color: colors[i]
        }));
        drawPieChart(canvas, data, { title: '按地形分布' });
    }

    return {
        drawBarChart,
        drawPieChart,
        drawFactorContribution,
        drawEraDistribution,
        drawTerrainDistribution,
        isOffscreenCanvasSupported: TerrainProfile.isOffscreenCanvasSupported,
        requestEfficientRender: TerrainProfile.requestEfficientRender,
        throttle: TerrainProfile.throttle,
        drawTerrainProfile: TerrainProfile.drawTerrainProfile,
        drawUncertaintyRing: TerrainProfile.drawUncertaintyRing,
        clearOffscreenCache: TerrainProfile.clearOffscreenCache
    };
})();
