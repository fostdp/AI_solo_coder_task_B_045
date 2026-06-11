const AppData = (function() {
    let battlefields = [];
    let filteredBattlefields = [];
    let roads = [];
    let rivers = [];
    let demGrid = [];
    let regions = [];
    let highProbAreas = [];
    let factors = [];
    let state = {};

    const ERA_COLORS = {
        '春秋战国': '#c0392b',
        '秦汉': '#e67e22',
        '三国两晋南北朝': '#f1c40f',
        '隋唐五代': '#27ae60',
        '宋辽金元': '#2980b9',
        '明清': '#8e44ad'
    };

    const TERRAIN_COLORS = {
        '山地': '#8b4513',
        '平原': '#228b22',
        '河谷': '#1e90ff',
        '关隘': '#ff6347'
    };

    function generateFallbackBattlefields() {
        const eras = ['春秋战国', '秦汉', '三国两晋南北朝', '隋唐五代', '宋辽金元', '明清'];
        const dynasties = ['春秋', '战国', '秦', '西汉', '东汉', '三国', '隋', '唐', '北宋', '南宋', '元', '明', '清'];
        const terrains = ['山地', '平原', '河谷', '关隘'];
        const results = ['A方胜', 'B方胜', '双方议和', '僵持不下'];
        const factions = ['秦', '楚', '齐', '燕', '赵', '魏', '汉', '晋', '隋', '唐', '宋', '元', '明', '清', '匈奴', '突厥', '吐蕃', '义军'];
        const prefixes = ['牧野', '长平', '巨鹿', '赤壁', '淝水', '官渡', '夷陵', '街亭', '雁门', '函谷', '虎牢', '襄阳', '彭城', '垓下', '定军', '祁山', '潼关', '井陉', '马陵', '桂陵'];
        const suffixes = ['之战', '大捷', '保卫战', '攻坚战', '伏击战'];

        const arr = [];
        for (let i = 0; i < 800; i++) {
            const lng = 73 + Math.random() * (135 - 73);
            const lat = 18 + Math.random() * (54 - 18);
            let elev = 100 + Math.abs(Math.sin(lng * 0.1) + Math.cos(lat * 0.1)) * 1000;
            if (lat > 40 && lng < 110) elev += 1000;
            if (lng < 95) elev += 2500;

            let terrain = terrains[Math.floor(Math.random() * terrains.length)];
            if (elev > 1500) terrain = '山地';
            else if (elev < 200 && lng > 110) terrain = '平原';

            const ta = 1000 + Math.floor(Math.random() * 499000);
            const tb = 1000 + Math.floor(Math.random() * 499000);

            arr.push({
                id: i + 1,
                battle_name: prefixes[Math.floor(Math.random() * prefixes.length)] + suffixes[Math.floor(Math.random() * suffixes.length)],
                dynasty: dynasties[Math.floor(Math.random() * dynasties.length)],
                era: eras[Math.floor(Math.random() * eras.length)],
                battle_year: -770 + Math.floor(Math.random() * 2682),
                belligerent_a: factions[Math.floor(Math.random() * factions.length)],
                belligerent_b: factions[Math.floor(Math.random() * factions.length)],
                troop_a: ta,
                troop_b: tb,
                total_troops: ta + tb,
                terrain_type: terrain,
                result: results[Math.floor(Math.random() * results.length)],
                lng, lat, elevation: elev,
                distance_to_river: 0.5 + Math.random() * 79.5,
                distance_to_road: 0.1 + Math.random() * 49.9
            });
        }
        return arr;
    }

    function generateFallbackRoads() {
        const roadTypes = ['驿道', '栈道', '漕运', '官道', '古道'];
        const dynasties = ['春秋', '战国', '秦', '汉', '隋', '唐', '宋', '元', '明', '清'];
        const arr = [];
        for (let i = 0; i < 60; i++) {
            const sl = 80 + Math.random() * 50;
            const sla = 25 + Math.random() * 25;
            const n = 5 + Math.floor(Math.random() * 10);
            const coords = [];
            for (let j = 0; j < n; j++) {
                coords.push([sl + j * 1.5 + Math.random() * 0.5, sla + Math.random() * 2 - 1]);
            }
            arr.push({
                id: i + 1,
                road_name: dynasties[i % dynasties.length] + '古道' + (i + 1) + '号',
                road_type: roadTypes[Math.floor(Math.random() * roadTypes.length)],
                dynasty: dynasties[Math.floor(Math.random() * dynasties.length)],
                importance: 1 + Math.floor(Math.random() * 5),
                coords
            });
        }
        return arr;
    }

    function generateFallbackRivers() {
        const riverNames = ['黄河', '长江', '淮河', '珠江', '海河', '辽河', '松花江', '汉江', '湘江', '赣江', '岷江', '嘉陵江', '大运河', '洞庭湖', '鄱阳湖', '太湖'];
        const arr = [];
        for (let i = 0; i < 25; i++) {
            const sl = 90 + Math.random() * 40;
            const sla = 25 + Math.random() * 20;
            const n = 5 + Math.floor(Math.random() * 15);
            const coords = [];
            for (let j = 0; j < n; j++) {
                coords.push([sl + j * 1.8, sla + Math.random() * 1.5 - 0.75]);
            }
            let rType = '河流';
            if (i > 18) rType = '湖泊';
            if (i === 19) rType = '运河';
            arr.push({
                id: i + 1,
                river_name: riverNames[i % riverNames.length],
                river_type: rType,
                coords
            });
        }
        return arr;
    }

    function generateFallbackDEM() {
        const cols = 50, rows = 40;
        const grid = [];
        for (let r = 0; r < rows; r++) {
            for (let c = 0; c < cols; c++) {
                const lng = 73 + c * (135 - 73) / (cols - 1);
                const lat = 54 - r * (54 - 18) / (rows - 1);
                let elev = 50 + Math.abs(Math.sin(lng * 0.05) + Math.cos(lat * 0.05)) * 800;
                if (lat > 40 && lng < 110) elev += 1000;
                if (lng < 95) elev += 2500;
                grid.push([lng, lat, elev]);
            }
        }
        return grid;
    }

    async function fetchJSON(url, fallbackFn) {
        try {
            const res = await fetch(url);
            if (res.ok) {
                return await res.json();
            }
        } catch (e) { }
        return fallbackFn();
    }

    async function loadAll() {
        const [bf, rd, rv, dm] = await Promise.all([
            fetchJSON('/api/battlefields', generateFallbackBattlefields),
            fetchJSON('/api/roads', generateFallbackRoads),
            fetchJSON('/api/rivers', generateFallbackRivers),
            fetchJSON('/api/dem', generateFallbackDEM)
        ]);
        battlefields = bf;
        roads = rd;
        rivers = rv;
        demGrid = dm;
        return { battlefields, roads, rivers, demGrid };
    }

    function getBattlefields() { return battlefields; }
    function getFilteredBattlefields() { return filteredBattlefields.length ? filteredBattlefields : battlefields; }
    function getRoads() { return roads; }
    function getRivers() { return rivers; }
    function getDEM() { return demGrid; }
    function getRegions() { return regions; }
    function getHighProbAreas() { return highProbAreas; }
    function getFactors() { return factors; }
    function getState() { return state; }

    function setRegions(r) { regions = r; }
    function setHighProbAreas(a) { highProbAreas = a; }
    function setFactors(f) { factors = f; }
    function setState(s) { state = Object.assign(state, s || {}); }
    function setFiltered(f) { filteredBattlefields = f || []; }

    function getEraColor(era) {
        return ERA_COLORS[era] || '#888888';
    }

    function getTerrainColor(t) {
        return TERRAIN_COLORS[t] || '#888888';
    }

    function getTroopSize(totalTroops) {
        if (totalTroops >= 500000) return 22;
        if (totalTroops >= 300000) return 18;
        if (totalTroops >= 150000) return 14;
        if (totalTroops >= 50000) return 11;
        if (totalTroops >= 10000) return 8;
        return 6;
    }

    function getEraColors() { return ERA_COLORS; }
    function getTerrainColors() { return TERRAIN_COLORS; }

    return {
        loadAll,
        getBattlefields,
        getFilteredBattlefields,
        getRoads,
        getRivers,
        getDEM,
        getRegions,
        getHighProbAreas,
        getFactors,
        getState,
        setRegions,
        setHighProbAreas,
        setFactors,
        setState,
        setFiltered,
        getEraColor,
        getTerrainColor,
        getTroopSize,
        getEraColors,
        getTerrainColors
    };
})();
