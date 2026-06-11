var BattleReplayer = (function () {
    var state = {
        data: null,
        currentFrame: 0,
        playing: false,
        timer: null,
        fps: 12,
        battlefieldId: null,
        activeBattlefield: null,
        replayLayers: [],
        eventMarkers: []
    };

    function formatHour(h) {
        var hours = Math.floor(h);
        var mins = Math.floor((h - hours) * 60);
        return 'T+' + pad(hours) + ':' + pad(mins);
    }

    function pad(n) { return n < 10 ? '0' + n : '' + n; }

    function loadReplay(bfId, bf, fps, callback) {
        if (state.battlefieldId === bfId && state.data) {
            if (callback) callback(state.data);
            return;
        }
        state.battlefieldId = bfId;
        state.activeBattlefield = bf;
        var url = '/api/battle_replay/' + bfId + '?fps=' + (fps || 12);
        fetchWithFallback(url, function () {
            return mockReplayData(bf, fps || 12);
        }, function (data) {
            state.data = data;
            state.currentFrame = 0;
            if (callback) callback(data);
        });
    }

    function mockReplayData(bf, fps) {
        var events = [];
        var types = ['部署', '进军', '交战', '伏击', '决战', '撤退'];
        var names = ['列阵待机', '主力推进', '两翼激战', '设伏合围', '决战冲阵', '交替掩护'];
        for (var i = 0; i < 6; i++) {
            events.push({
                id: i + 1,
                event_order: i + 1,
                event_type: types[i],
                event_name: names[i],
                description: '模拟战役事件描述',
                hour_offset: i * 4,
                lng: bf.lng + (Math.random() - 0.5) * 0.2,
                lat: bf.lat + (Math.random() - 0.5) * 0.15,
                belligerent: i % 2 === 0 ? bf.belligerent_a : bf.belligerent_b,
                troop_count: Math.floor(bf.total_troops / 4),
                casualties: Math.floor(bf.total_troops / 20),
                is_turning_point: i === 3,
                is_decision: i === 2,
                tags: [types[i]],
                nlp_confidence: 0.75 + Math.random() * 0.2
            });
        }
        var totalFrames = fps * 8;
        var frames = [];
        for (var f = 0; f < totalFrames; f++) {
            var t = f / (totalFrames - 1);
            var ts = t * 24;
            var positions = [];
            var unitsA = ['左军', '中军', '右军'];
            var unitsB = ['左翼', '主力', '右翼'];
            for (var k = 0; k < 3; k++) {
                positions.push({
                    belligerent: bf.belligerent_a,
                    unit_name: unitsA[k],
                    lng: bf.lng - 0.15 + t * 0.08 + Math.sin(t * Math.PI + k) * 0.02,
                    lat: bf.lat + (k - 1) * 0.06,
                    troop_count: Math.floor(bf.troop_a / 3 * (1 - t * 0.3)),
                    icon_type: 'infantry'
                });
                positions.push({
                    belligerent: bf.belligerent_b,
                    unit_name: unitsB[k],
                    lng: bf.lng + 0.15 - t * 0.06 + Math.sin(t * Math.PI + 0.5 + k) * 0.02,
                    lat: bf.lat + (k - 1) * 0.06,
                    troop_count: Math.floor(bf.troop_b / 3 * (1 - t * 0.2)),
                    icon_type: k % 2 === 0 && t > 0.3 ? 'cavalry' : 'infantry'
                });
            }
            var ae = null;
            for (var ei = 0; ei < events.length; ei++) {
                if (ts >= events[ei].hour_offset && ts < events[ei].hour_offset + 1.5) {
                    ae = events[ei];
                    break;
                }
            }
            frames.push({
                frame_index: f,
                timestamp_h: ts,
                time_label: formatHour(ts),
                troop_positions: positions,
                active_event: ae,
                front_lines: t > 0.2 && t < 0.85 ? [buildFrontLine(bf, t)] : []
            });
        }
        return {
            timeline: {
                battlefield_id: bf.id,
                battle_name: bf.battle_name,
                total_duration_h: 24,
                events: events,
                turning_points: events.filter(function (e) { return e.is_turning_point; }),
                decisions: events.filter(function (e) { return e.is_decision; })
            },
            frames: frames,
            fps: fps,
            total_frames: totalFrames,
            nlp_stats: {
                total_events: events.length,
                avg_confidence: 0.85,
                turning_point_count: events.filter(function (e) { return e.is_turning_point; }).length,
                decision_count: events.filter(function (e) { return e.is_decision; }).length
            }
        };
    }

    function buildFrontLine(bf, t) {
        var line = [];
        for (var i = 0; i < 8; i++) {
            line.push([
                bf.lng + t * 0.02 + Math.sin(i * 0.8 + t * Math.PI) * 0.015,
                bf.lat - 0.2 + i * 0.057
            ]);
        }
        return line;
    }

    function fetchWithFallback(url, fallbackFn, callback) {
        fetch(url).then(function (r) {
            if (!r.ok) throw new Error('fail');
            return r.json();
        }).then(callback).catch(function () {
            if (fallbackFn) callback(fallbackFn());
        });
    }

    function clearReplayLayers(map) {
        for (var i = 0; i < state.replayLayers.length; i++) {
            map.removeLayer(state.replayLayers[i]);
        }
        state.replayLayers = [];
        for (var j = 0; j < state.eventMarkers.length; j++) {
            map.removeLayer(state.eventMarkers[j]);
        }
        state.eventMarkers = [];
    }

    function renderFrame(map, frame) {
        clearReplayLayers(map);

        for (var i = 0; i < frame.troop_positions.length; i++) {
            var pos = frame.troop_positions[i];
            var colorA = '#3498db';
            var colorB = '#e74c3c';
            var color = pos.belligerent === state.activeBattlefield.belligerent_a ? colorA : colorB;
            var size = Math.max(8, Math.min(22, 8 + pos.troop_count / 8000));
            var iconHtml = pos.icon_type === 'cavalry'
                ? '<div style="background:' + color + ';width:' + size + 'px;height:' + size + 'px;border-radius:50%;border:2px solid #f4e5b0;display:flex;align-items:center;justify-content:center;color:#fff;font-size:10px;font-weight:bold">骑</div>'
                : '<div style="background:' + color + ';width:' + size + 'px;height:' + size + 'px;clip-path:polygon(50% 0%,0% 100%,100% 100%);border-bottom:2px solid #f4e5b0"></div>';
            var marker = L.marker([pos.lat, pos.lng], {
                icon: L.divIcon({
                    html: iconHtml,
                    className: 'unit-marker',
                    iconSize: [size, size],
                    iconAnchor: [size / 2, size / 2]
                })
            }).bindTooltip(pos.belligerent + ' - ' + pos.unit_name + ': ' + pos.troop_count.toLocaleString() + '人');
            marker.addTo(map);
            state.replayLayers.push(marker);
        }

        for (var fl = 0; fl < frame.front_lines.length; fl++) {
            var line = [];
            for (var k = 0; k < frame.front_lines[fl].length; k++) {
                line.push([frame.front_lines[fl][k][1], frame.front_lines[fl][k][0]]);
            }
            var pline = L.polyline(line, {
                color: '#d4af37',
                weight: 3,
                opacity: 0.8,
                dashArray: '8,4'
            }).addTo(map);
            state.replayLayers.push(pline);
        }

        if (frame.active_event) {
            var ev = frame.active_event;
            var evColor = ev.is_turning_point ? '#e74c3c' : (ev.is_decision ? '#3498db' : '#f39c12');
            var evMarker = L.circleMarker([ev.lat, ev.lng], {
                radius: 14,
                fillColor: evColor,
                color: '#f4e5b0',
                weight: 3,
                opacity: 1,
                fillOpacity: 0.9
            }).bindTooltip('<b>' + ev.event_name + '</b><br/>' + ev.event_type + ' - ' + formatHour(ev.hour_offset) + '<br/>' + ev.description, { direction: 'top' }).openTooltip();
            evMarker.addTo(map);
            state.replayLayers.push(evMarker);
        }
    }

    function renderEventMarkers(map, events) {
        for (var j = 0; j < state.eventMarkers.length; j++) {
            map.removeLayer(state.eventMarkers[j]);
        }
        state.eventMarkers = [];
        for (var i = 0; i < events.length; i++) {
            var ev = events[i];
            var evColor = ev.is_turning_point ? '#e74c3c' : (ev.is_decision ? '#3498db' : '#95a5a6');
            var m = L.circleMarker([ev.lat, ev.lng], {
                radius: 6,
                fillColor: evColor,
                color: '#fff',
                weight: 2,
                fillOpacity: 0.85
            }).bindTooltip(formatHour(ev.hour_offset) + ' ' + ev.event_name);
            m.addTo(map);
            state.eventMarkers.push(m);
        }
    }

    function renderEventList(events) {
        var el = document.getElementById('replay-event-list');
        if (!el) return;
        el.innerHTML = '';
        for (var i = 0; i < events.length; i++) {
            var ev = events[i];
            var cls = 'event-item';
            if (ev.is_turning_point) cls += ' turning-point';
            if (ev.is_decision) cls += ' decision';
            var div = document.createElement('div');
            div.className = cls;
            div.innerHTML = '<span class="event-time">' + formatHour(ev.hour_offset) + '</span>' +
                '<span class="event-name">' + ev.event_name + '</span>' +
                '<span class="event-type-tag">' + ev.event_type + '</span>' +
                (ev.is_turning_point ? '<span class="event-type-tag" style="background:#e74c3c;color:#fff">转折点</span>' : '') +
                (ev.is_decision ? '<span class="event-type-tag" style="background:#3498db;color:#fff">决策</span>' : '');
            (function (idx) {
                div.addEventListener('click', function () {
                    if (!state.data) return;
                    var ev2 = state.data.timeline.events[idx];
                    var targetTs = ev2.hour_offset;
                    var duration = state.data.timeline.total_duration_h || 24;
                    var targetFrame = Math.floor((targetTs / duration) * (state.data.frames.length - 1));
                    state.currentFrame = Math.max(0, Math.min(state.data.frames.length - 1, targetFrame));
                    updateUI();
                });
            })(i);
            el.appendChild(div);
        }
    }

    function updateUI() {
        if (!state.data) return;
        var frame = state.data.frames[state.currentFrame];
        document.getElementById('replay-time-label').textContent = frame.time_label;
        document.getElementById('replay-event-name').textContent = frame.active_event ? frame.active_event.event_name : '-';
        document.getElementById('replay-timeline').value = state.currentFrame;
        document.getElementById('replay-timeline').max = state.data.frames.length - 1;
    }

    function startPlay(map) {
        if (state.playing || !state.data) return;
        state.playing = true;
        document.getElementById('btn-replay-play').textContent = '⏸';
        var interval = 1000 / state.fps;
        state.timer = setInterval(function () {
            state.currentFrame++;
            if (state.currentFrame >= state.data.frames.length) {
                state.currentFrame = 0;
            }
            renderFrame(map, state.data.frames[state.currentFrame]);
            updateUI();
        }, interval);
    }

    function stopPlay(map) {
        state.playing = false;
        document.getElementById('btn-replay-play').textContent = '▶';
        if (state.timer) {
            clearInterval(state.timer);
            state.timer = null;
        }
    }

    function stepPrev(map) {
        if (!state.data) return;
        state.currentFrame = Math.max(0, state.currentFrame - 1);
        renderFrame(map, state.data.frames[state.currentFrame]);
        updateUI();
    }

    function stepNext(map) {
        if (!state.data) return;
        state.currentFrame = Math.min(state.data.frames.length - 1, state.currentFrame + 1);
        renderFrame(map, state.data.frames[state.currentFrame]);
        updateUI();
    }

    function initUI(map, bf, callback) {
        state.fps = parseInt(document.getElementById('replay-fps').value) || 12;
        document.getElementById('replay-timeline-container').classList.remove('hidden');
        loadReplay(bf.id, bf, state.fps, function (data) {
            state.currentFrame = 0;
            renderEventMarkers(map, data.timeline.events);
            renderEventList(data.timeline.events);
            renderFrame(map, data.frames[0]);
            updateUI();
            if (callback) callback(data);
        });

        document.getElementById('btn-replay-play').onclick = function () {
            if (state.playing) stopPlay(map); else startPlay(map);
        };
        document.getElementById('btn-replay-prev').onclick = function () { stepPrev(map); };
        document.getElementById('btn-replay-next').onclick = function () { stepNext(map); };
        document.getElementById('replay-timeline').addEventListener('input', function (e) {
            if (!state.data) return;
            stopPlay(map);
            state.currentFrame = parseInt(e.target.value);
            renderFrame(map, state.data.frames[state.currentFrame]);
            updateUI();
        });
        document.getElementById('replay-fps').addEventListener('change', function (e) {
            state.fps = parseInt(e.target.value) || 12;
            if (state.playing) { stopPlay(map); startPlay(map); }
        });
    }

    function reset(map) {
        stopPlay(map);
        clearReplayLayers(map);
        state.data = null;
        state.battlefieldId = null;
        state.activeBattlefield = null;
        document.getElementById('replay-timeline-container').classList.add('hidden');
    }

    function getEventsForBattle(bfId, bf, callback) {
        fetchWithFallback('/api/battle_events/' + bfId, function () {
            return { events: mockReplayData(bf, 12).timeline.events, event_count: 6 };
        }, function (data) { if (callback) callback(data); });
    }

    return {
        initUI: initUI,
        reset: reset,
        startPlay: startPlay,
        stopPlay: stopPlay,
        stepPrev: stepPrev,
        stepNext: stepNext,
        renderFrame: renderFrame,
        clearReplayLayers: clearReplayLayers,
        getEventsForBattle: getEventsForBattle,
        formatHour: formatHour
    };
})();