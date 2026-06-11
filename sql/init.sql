-- 古代战场遗址空间分布与军事地理分析系统 - 数据库初始化脚本
-- PostgreSQL + PostGIS

-- 启用PostGIS扩展
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_raster;

-- ==================== 战场遗址表 ====================
DROP TABLE IF EXISTS battlefield CASCADE;
CREATE TABLE battlefield (
    id SERIAL PRIMARY KEY,
    battle_name VARCHAR(200) NOT NULL,
    dynasty VARCHAR(100) NOT NULL,
    era VARCHAR(50) NOT NULL,
    battle_year INT NOT NULL,
    belligerent_a VARCHAR(200) NOT NULL,
    belligerent_b VARCHAR(200) NOT NULL,
    troop_a INT NOT NULL DEFAULT 0,
    troop_b INT NOT NULL DEFAULT 0,
    total_troops INT NOT NULL DEFAULT 0,
    terrain_type VARCHAR(20) NOT NULL CHECK (terrain_type IN ('山地', '平原', '河谷', '关隘')),
    result VARCHAR(200) NOT NULL,
    geom GEOMETRY(Point, 4326) NOT NULL,
    elevation NUMERIC(10, 2) DEFAULT 0,
    distance_to_river NUMERIC(10, 2) DEFAULT 0,
    distance_to_road NUMERIC(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_battlefield_geom ON battlefield USING GIST(geom);
CREATE INDEX idx_battlefield_era ON battlefield(era);
CREATE INDEX idx_battlefield_terrain ON battlefield(terrain_type);
CREATE INDEX idx_battlefield_total_troops ON battlefield(total_troops);

-- ==================== 古代交通道路表 ====================
DROP TABLE IF EXISTS ancient_road CASCADE;
CREATE TABLE ancient_road (
    id SERIAL PRIMARY KEY,
    road_name VARCHAR(200) NOT NULL,
    road_type VARCHAR(50) NOT NULL DEFAULT '驿道' CHECK (road_type IN ('驿道', '栈道', '漕运', '官道', '古道')),
    dynasty VARCHAR(100) NOT NULL,
    importance INT NOT NULL DEFAULT 1 CHECK (importance BETWEEN 1 AND 5),
    geom GEOMETRY(LineString, 4326) NOT NULL
);

CREATE INDEX idx_ancient_road_geom ON ancient_road USING GIST(geom);
CREATE INDEX idx_ancient_road_dynasty ON ancient_road(dynasty);

-- ==================== 河流水系表 ====================
DROP TABLE IF EXISTS ancient_river CASCADE;
CREATE TABLE ancient_river (
    id SERIAL PRIMARY KEY,
    river_name VARCHAR(200) NOT NULL,
    river_type VARCHAR(50) NOT NULL DEFAULT '河流' CHECK (river_type IN ('河流', '湖泊', '运河')),
    geom GEOMETRY(LineString, 4326) NOT NULL
);

CREATE INDEX idx_ancient_river_geom ON ancient_river USING GIST(geom);

-- ==================== DEM地形栅格数据表 ====================
DROP TABLE IF EXISTS dem_tile CASCADE;
CREATE TABLE dem_tile (
    id SERIAL PRIMARY KEY,
    tile_x INT NOT NULL,
    tile_y INT NOT NULL,
    zoom INT NOT NULL,
    rast RASTER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dem_tile_rast ON dem_tile USING GIST(ST_ConvexHull(rast));
CREATE UNIQUE INDEX idx_dem_tile_xyz ON dem_tile(tile_x, tile_y, zoom);

-- ==================== 军事地理分区表 ====================
DROP TABLE IF EXISTS military_region CASCADE;
CREATE TABLE military_region (
    id SERIAL PRIMARY KEY,
    region_name VARCHAR(200) NOT NULL,
    region_code VARCHAR(50) NOT NULL,
    battle_count INT NOT NULL DEFAULT 0,
    avg_density NUMERIC(10, 4) NOT NULL DEFAULT 0,
    dominant_terrain VARCHAR(50),
    geom GEOMETRY(Polygon, 4326) NOT NULL
);

CREATE INDEX idx_military_region_geom ON military_region USING GIST(geom);

-- ==================== 高概率战场区域表 ====================
DROP TABLE IF EXISTS high_prob_area CASCADE;
CREATE TABLE high_prob_area (
    id SERIAL PRIMARY KEY,
    probability NUMERIC(5, 4) NOT NULL CHECK (probability BETWEEN 0 AND 1),
    terrain_factor NUMERIC(5, 4) NOT NULL DEFAULT 0,
    road_factor NUMERIC(5, 4) NOT NULL DEFAULT 0,
    river_factor NUMERIC(5, 4) NOT NULL DEFAULT 0,
    geom GEOMETRY(Polygon, 4326) NOT NULL
);

CREATE INDEX idx_high_prob_area_geom ON high_prob_area USING GIST(geom);
CREATE INDEX idx_high_prob_area_prob ON high_prob_area(probability);

-- ==================== 选址影响因素分析结果表 ====================
DROP TABLE IF EXISTS site_selection_factor CASCADE;
CREATE TABLE site_selection_factor (
    id SERIAL PRIMARY KEY,
    factor_name VARCHAR(100) NOT NULL,
    contribution NUMERIC(5, 4) NOT NULL DEFAULT 0,
    p_value NUMERIC(10, 6) NOT NULL DEFAULT 1,
    odds_ratio NUMERIC(10, 4) NOT NULL DEFAULT 1,
    method VARCHAR(50) NOT NULL DEFAULT '逻辑回归',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ==================== 战役事件表（战役进程时空回放） ====================
DROP TABLE IF EXISTS battle_event CASCADE;
CREATE TABLE battle_event (
    id SERIAL PRIMARY KEY,
    battlefield_id INT NOT NULL REFERENCES battlefield(id) ON DELETE CASCADE,
    event_order INT NOT NULL,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('部署', '进军', '交战', '伏击', '突围', '撤退', '决策', '补给', '决战', '其他')),
    event_name VARCHAR(200) NOT NULL,
    description TEXT,
    hour_offset NUMERIC(8, 2) NOT NULL DEFAULT 0,
    geom GEOMETRY(Point, 4326),
    belligerent VARCHAR(100),
    troop_count INT DEFAULT 0,
    casualties INT DEFAULT 0,
    is_turning_point BOOLEAN DEFAULT FALSE,
    is_decision BOOLEAN DEFAULT FALSE,
    tags TEXT[],
    extracted_from TEXT,
    nlp_confidence NUMERIC(5, 4) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_battle_event_bfid ON battle_event(battlefield_id);
CREATE INDEX idx_battle_event_geom ON battle_event USING GIST(geom);
CREATE INDEX idx_battle_event_order ON battle_event(battlefield_id, event_order);

-- ==================== 后勤补给节点与路线表 ====================
DROP TABLE IF EXISTS supply_node CASCADE;
CREATE TABLE supply_node (
    id SERIAL PRIMARY KEY,
    node_name VARCHAR(200) NOT NULL,
    node_type VARCHAR(50) NOT NULL CHECK (node_type IN ('粮仓', '武库', '兵站', '渡口', '驿站', '关卡', '集散地')),
    belligerent VARCHAR(100) NOT NULL,
    geom GEOMETRY(Point, 4326),
    capacity INT DEFAULT 0,
    is_bottleneck BOOLEAN DEFAULT FALSE,
    throughput NUMERIC(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_supply_node_geom ON supply_node USING GIST(geom);
CREATE INDEX idx_supply_node_belligerent ON supply_node(belligerent);

DROP TABLE IF EXISTS supply_route CASCADE;
CREATE TABLE supply_route (
    id SERIAL PRIMARY KEY,
    route_name VARCHAR(200) NOT NULL,
    belligerent VARCHAR(100) NOT NULL,
    geom GEOMETRY(LineString, 4326),
    total_length_km NUMERIC(10, 2) DEFAULT 0,
    capacity INT DEFAULT 0,
    est_time_days NUMERIC(8, 2) DEFAULT 0,
    efficiency NUMERIC(5, 4) DEFAULT 0,
    bottleneck_ids INT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_supply_route_geom ON supply_route USING GIST(geom);
CREATE INDEX idx_supply_route_belligerent ON supply_route(belligerent);

-- ==================== 军事工程与防御盲区表 ====================
DROP TABLE IF EXISTS military_structure CASCADE;
CREATE TABLE military_structure (
    id SERIAL PRIMARY KEY,
    structure_name VARCHAR(200) NOT NULL,
    structure_type VARCHAR(50) NOT NULL CHECK (structure_type IN ('城墙', '关隘', '堡垒', '烽火台', '要塞', '寨堡', '护城河')),
    battlefield_id INT REFERENCES battlefield(id) ON DELETE SET NULL,
    dynasty VARCHAR(100),
    geom GEOMETRY(Point, 4326),
    polygon GEOMETRY(Polygon, 4326),
    height_m NUMERIC(8, 2) DEFAULT 0,
    length_m NUMERIC(10, 2) DEFAULT 0,
    thickness_m NUMERIC(8, 2) DEFAULT 0,
    material VARCHAR(50),
    gate_count INT DEFAULT 0,
    tower_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_military_structure_geom ON military_structure USING GIST(geom);
CREATE INDEX idx_military_structure_polygon ON military_structure USING GIST(polygon);
CREATE INDEX idx_military_structure_type ON military_structure(structure_type);

DROP TABLE IF EXISTS defense_blind_zone CASCADE;
CREATE TABLE defense_blind_zone (
    id SERIAL PRIMARY KEY,
    structure_id INT NOT NULL REFERENCES military_structure(id) ON DELETE CASCADE,
    center_geom GEOMETRY(Point, 4326),
    area_geom GEOMETRY(Polygon, 4326),
    area_km2 NUMERIC(10, 4) DEFAULT 0,
    direction VARCHAR(10),
    max_distance_km NUMERIC(8, 2) DEFAULT 0,
    visibility_pct NUMERIC(5, 4) DEFAULT 0,
    risk_level VARCHAR(10) CHECK (risk_level IN ('低', '中', '高', '极高')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_defense_blind_zone_structure ON defense_blind_zone(structure_id);
CREATE INDEX idx_defense_blind_zone_geom ON defense_blind_zone USING GIST(area_geom);

-- ==================== 军事思想演变节点表 ====================
DROP TABLE IF EXISTS doctrine_evolution CASCADE;
CREATE TABLE doctrine_evolution (
    id SERIAL PRIMARY KEY,
    year INT NOT NULL,
    era_boundary VARCHAR(100),
    before_doctrine VARCHAR(100),
    after_doctrine VARCHAR(100),
    change_magnitude NUMERIC(5, 4) DEFAULT 0,
    confidence NUMERIC(5, 4) DEFAULT 0,
    key_features TEXT[],
    trigger_events TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_doctrine_evolution_year ON doctrine_evolution(year);
