const baseurl = "<insert_base_url_here>";
const apiKey = new URLSearchParams(document.location.search).get("api-key");

// Leaflet map
let map;

// hashmap of trackers
const trackers = new Map();

// Playback state
let playback = {
    trackerId: null,
    latlngs: [],
    rawPoints: [],
    index: 0,
    playing: false,
    timerId: null,
    speed: 1, // 1x, 2x, 4x
    marker: null
};

/**
* Handle clicks for the show On Map checkbox for each tracker
* @param {HTMLInputElement} checkbox - The affected checkbox
*/
function onTrackerVisibilityToggle(checkbox){
    const id = checkbox.closest(".tracker-card").dataset.id;
    const tracker = trackers.get(id);
    setTrackerVisibility(tracker,checkbox.checked);
}

/**
* Set the visibility for all trackers
* @param {boolean} visible - Visiblity of all tracker to set
*/
function setAllTrackersVisibility(visible){
    trackers.forEach((tracker) => setTrackerVisibility(tracker, visible));
}

/**
* Set the visibility for a single tracker
* @param {Object} tracker - Tracker to set visibility of
* @param {boolean} visible - Visiblity to set tracker to
    */
function setTrackerVisibility(tracker,visible){
    // Add/Remove marker if visibility has changed
    if (tracker.Visible != visible){
        tracker.Visible = visible;
        if (tracker.Visible){
            tracker.Marker.addTo(map);
            if (tracker.Polyline){ tracker.Polyline.addTo(map); }
        } else {
            tracker.Marker.remove();
            if (tracker.Polyline){ tracker.Polyline.remove(); }
        }
    }

    updateTrackerCardCheckbox(tracker,visible);
}

/**
 * Ensure the visibility toggle is set to visiblity value
 * @param {Object} tracker - Tracker of checkbox to change
 * @param {boolean} checked - Checked state to set checkbox to
*/
function updateTrackerCardCheckbox(tracker, checked){
    // Get trackerCard by data-id
    const trackerCard = document.querySelector(`[data-id~="${tracker.Id}"]`);
    // Get checkbox in card
    const checkbox = trackerCard.getElementsByTagName("input")[0];
    checkbox.checked = checked;
}

/**
 * Handle go to tracker click event
 * @param {HTMLButtonElement} button - Button clicked 
*/
function handleGoToTracker(button){
    const id = button.closest(".tracker-card").dataset.id;
    let t = trackers.get(id);
    if (t.Lat == 0){
        window.alert("There is no available location data for selected tracker")
    }
    else {
        // history range selection
        const windowMs = getSelectedRangeWindowMs();
        if (windowMs === 0){
            stopPlayback();
            zoomToTracker(t);
            return;
        }
        showHistoryAndAnimateByWindow(t, windowMs);
    }

}

/**
 * Zoom/pan to a tracker
 * @param {Object} tracker - Tracker to zoom/pan to 
*/
function zoomToTracker(tracker){
    map.flyTo([tracker.Lat,tracker.Lon],14);
}


/**
 * reset trackers list, create fragment with all tracker cards, and then append to container
*/
function renderTrackersList(){
    const container = document.getElementById("trackersList");
    container.innerHTML = "";
    const fragment = document.createDocumentFragment();
    trackers.values().forEach(tracker => {
        fragment.appendChild(createTrackerCard(tracker));
    });
    container.appendChild(fragment);
}

/**
 * Create a tailwind card for a tracker
 * The card design and layout is based on https://tailwindflex.com/@odessa-young/stacked-list
 * @param {Object} tracker - The tracker to make a card of
 * @returns {HTMLElement} - The tracker card element
 */
function createTrackerCard(tracker){
    const card = document.createElement("div");
    card.className = "tracker-card bg-white border border-gray-200 shadow-sm dark:bg-gray-800 dark:border-gray-700 rounded-md shadow transition transform hover:shadow-lg hover:-translate-y-0.5 p-4"
    card.dataset.id = tracker.Id;
    card.innerHTML = `
<div class="flex items-center justify-between">
    <div>
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">Name: ${tracker.Name}</h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">ID/IMEI: ${tracker.Id}</p>
    </div>
</div>
<div class="mt-4 flex items-center justify-between">
    <p class="text-sm font-medium text-gray-500 dark:text-gray-400">
        Last position update: ${formatTimestamp(tracker.Timestamp)}
    </p>
    <a
        onclick = "handleGoToTracker(this)"
        class="flex items-center text-indigo-600 text-sm font-medium hover:underline focus:outline-none"
        href="#">
        <svg class="h-4 w-4 mr-1" fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
            <path d="M10 3a1 1 0 011 1v6h6a1 1 0 110 2h-6v6a1 1 0 11-2 0v-6H3a1 1 0 110-2h6V4a1 1 0 011-1z"/>
        </svg>
        Go to tracker
    </a>
</div>
<label class="mt-4 flex items-center text-sm text-gray-700 dark:text-gray-300">
    <input
        type="checkbox"
        onchange="onTrackerVisibilityToggle(this)"
        class="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
        ${tracker.Visible ? "checked" : ""}
    />
    <span class="ml-2">Show on map</span>
</label>
</div>`;
    return card;
}


/**
 * Update location properties of a tracker, and then equivalently 
 * update the marker and sidebar card 
 * @param {Object} tracker - Tracker to update values of 
 * @param {Object} newData - Tracker object with updated location data
 */
function updateTrackerLocation(tracker,newData){
    tracker.Timestamp = newData.Timestamp;
    tracker.Lat = newData.Lat/2000000;
    tracker.Lon = newData.Lon/2000000;
    tracker.Speed = newData.Speed;
    tracker.Heading = newData.Heading;
    updateMarker(tracker);
    updateSidebarTable(tracker);
}

/**
 * Update timestamp on tracker card 
 * The card design and layout is based on https://tailwindflex.com/@odessa-young/stacked-list
 * @param {Object} tracker - The tracker to make a card of
 */
function updateSidebarTable(tracker){
    const row = document.querySelector(`div[data-id="${tracker.Id}"]`);
    if (row != null){
        row.querySelectorAll('p')[1].textContent = "Last position update: " + formatTimestamp(tracker.Timestamp);
    }
}

/**
 * Convert date from RFC3339 to locale format
 * @param {string} date - Date as RFC3339 string
 * @returns {string} - The locale string
 */
function formatTimestamp(date){
    return new Date(date).toLocaleString("da-DK").replace(" ","");
}

/** 
 * Update the coordinates and popup message of a tracker/marker
 * @param {Object} tracker - Tracker to update marker of
*/
function updateMarker(tracker){
    tracker.Marker.setLatLng([tracker.Lat,tracker.Lon])
    const timestamp = formatTimestamp(tracker.Timestamp);
    tracker.Marker.bindPopup("<b>Timestamp: "+timestamp+ "</br>ID: "+tracker.Id+"</br>Name: "+tracker.Name+"</br>Model: "+tracker.Model+"</b>");
}

/**
 * Fetch last N locations and render a polyline for a tracker
 * @param {Object} tracker
 * @param {number} limit
 */
async function renderTrackHistory(tracker, limit=50){
    try{
        const data = await apiGet(`trackers/${tracker.Id}/locations?limit=${limit}`);
        if (!Array.isArray(data) || data.length === 0){ return; }
        const latlngs = data.map(p => [p.Lat/2000000, p.Lon/2000000]);
        if (latlngs.length < 2){ return; }
        // remove existing polyline
        if (tracker.Polyline){ tracker.Polyline.remove(); }
        tracker.Polyline = L.polyline(latlngs, {color:'#2563eb', weight:3, opacity:0.9});
        if (tracker.Visible){ tracker.Polyline.addTo(map); }
        const bounds = tracker.Polyline.getBounds();
        if (bounds && bounds.isValid()){ map.fitBounds(bounds.pad(0.2)); }
    } catch(e){
        console.warn('Failed to render history for', tracker.Id, e);
    }
}

/**
 * Store trackers in global variable, create markers for each 
 * @param {Array} data - List af tracker data from API
 */
async function initiateTrackers(data){
    const markericon = new L.Icon.Default();
    markericon.options.shadowSize = [0,0];
    data.forEach((tracker) => {
        tracker.Visible = true;
        tracker.Polyline = null;
        tracker.Marker = L.marker([0,0], {icon: markericon});
        trackers.set(tracker.Id,tracker);  
        updateTrackerLocation(tracker,tracker)
        tracker.Marker.addTo(map)
    });
}

/** Map history ranges to window size in milliseconds */
function getSelectedRangeWindowMs(){
    const sel = document.getElementById('historyRange');
    const v = sel ? sel.value : 'off';
    switch(v){
        case '15m': return 15*60*1000;
        case '1h': return 60*60*1000;
        case '6h': return 6*60*60*1000;
        case '24h': return 24*60*60*1000;
        case 'off':
        default: return 0;
    }
}

/** Fetch, render history and start playback for a tracker using a time window (ms) ending now */
async function showHistoryAndAnimateByWindow(tracker, windowMs){
    try{
        const end = new Date();
        const start = new Date(end.getTime() - windowMs);
        const params = new URLSearchParams({ start: start.toISOString(), end: end.toISOString() });
        const data = await apiGet(`trackers/${tracker.Id}/locations?${params.toString()}`);
        if (!Array.isArray(data) || data.length < 2){
            window.alert('Not enough history data to play route');
            return;
        }
        const latlngs = data.map(p => [p.Lat/2000000, p.Lon/2000000]);
        // Render polyline
        if (tracker.Polyline){ tracker.Polyline.remove(); }
        tracker.Polyline = L.polyline(latlngs, {color:'#6366f1', weight:4, opacity:0.6});
        if (tracker.Visible){ tracker.Polyline.addTo(map); }
        const bounds = tracker.Polyline.getBounds();
        if (bounds && bounds.isValid()){ map.fitBounds(bounds.pad(0.15)); }
        // Start playback
        startPlayback(tracker, latlngs, data);
    } catch(e){
        console.warn('Failed to load history for', tracker.Id, e);
        window.alert('Failed to load history');
    }
}

function startPlayback(tracker, latlngs, rawPoints){
    stopPlayback(); // ensure only one playback at a time
    playback.trackerId = tracker.Id;
    playback.latlngs = latlngs;
    playback.rawPoints = rawPoints;
    playback.index = 0;
    playback.playing = true;
    playback.speed = Number(document.getElementById('playbackSpeed')?.value || 1);
    // Create or reuse marker
    const start = latlngs[0];
    playback.marker = L.circleMarker(start, {radius:6, color:'#1f2937', weight:2, fillColor:'#22c55e', fillOpacity:0.9});
    if (tracker.Visible){ playback.marker.addTo(map); }
    // Show panel
    updatePlaybackPanel(true, tracker);
    // Hide other trackers' polylines while this playback is active
    hideOtherTrackersPolylines(tracker.Id);
    scheduleNextFrame();
}

function scheduleNextFrame(){
    clearTimeout(playback.timerId);
    if (!playback.playing) return;
    const stepMs = Math.max(16, 60 - (playback.speed*10)); // rough speed scaler
    playback.timerId = setTimeout(playbackFrame, stepMs);
}

function playbackFrame(){
    if (!playback.playing) return;
    if (playback.index >= playback.latlngs.length - 1){
        // Reached end
        playback.playing = false;
        updatePlayPauseButton();
        return;
    }
    stepToIndex(playback.index + 1);
    scheduleNextFrame();
}

function togglePlayback(){
    if (!playback.latlngs.length) return;
    playback.playing = !playback.playing;
    updatePlayPauseButton();
    if (playback.playing){ scheduleNextFrame(); }
}

function stopPlayback(){
    clearTimeout(playback.timerId);
    if (playback.marker){ playback.marker.remove(); }
    // Restore any previously hidden polylines
    restoreHiddenPolylines();
    playback = { trackerId: null, latlngs: [], rawPoints: [], index: 0, playing: false, timerId: null, speed: 1, marker: null };
    updatePlaybackPanel(false);
}

function scrubPlayback(percent){
    if (!playback.latlngs.length) return;
    const idx = Math.round((Number(percent)/100) * (playback.latlngs.length-1));
    stepToIndex(idx);
}

function setPlaybackSpeed(v){
    playback.speed = Number(v) || 1;
}

function updatePlaybackPanel(show, tracker){
    const panel = document.getElementById('playbackPanel');
    if (!panel) return;
    if (show){
        panel.classList.remove('hidden');
        const label = document.getElementById('playbackLabel');
        if (label && tracker){ label.textContent = `Route playback – ${tracker.Name || tracker.Id}`; }
        updatePlayPauseButton();
        const slider = document.getElementById('playbackProgress');
        if (slider){ slider.value = '0'; }
    } else {
        panel.classList.add('hidden');
    }
}

function updatePlayPauseButton(){
    const btn = document.getElementById('btnPlayPause');
    if (!btn) return;
    btn.textContent = playback.playing ? 'Pause' : 'Play';
}

function stepToIndex(newIndex){
    const clamped = Math.max(0, Math.min(playback.latlngs.length-1, newIndex));
    playback.index = clamped;
    const p = playback.latlngs[playback.index];
    if (playback.marker){ playback.marker.setLatLng(p); }
    // Update slider
    const progress = Math.round((playback.index/(playback.latlngs.length-1))*100);
    const slider = document.getElementById('playbackProgress');
    if (slider){ slider.value = String(progress); }
    // Update point info (timestamp + speed/heading if present)
    const info = document.getElementById('pointInfo');
    if (info && playback.rawPoints.length){
        const r = playback.rawPoints[playback.index];
        // Force 24h format using locale options (da-DK already is 24h, but being explicit).
        const ts = r.Timestamp ? new Date(r.Timestamp).toLocaleString('da-DK', { hour12: false }) : '';
        const sp = r.Speed != null ? ` • speed: ${r.Speed}` : '';
        const hd = r.Heading != null ? ` • heading: ${r.Heading}` : '';
        info.textContent = `Point ${playback.index+1}/${playback.latlngs.length} • ${ts}${sp}${hd}`;
    }
}

function stepForward(){
    if (!playback.latlngs.length) return;
    // Pausing ensures deterministic stepping
    if (playback.playing){ playback.playing = false; updatePlayPauseButton(); }
    stepToIndex(playback.index + 1);
}

function stepBack(){
    if (!playback.latlngs.length) return;
    if (playback.playing){ playback.playing = false; updatePlayPauseButton(); }
    stepToIndex(playback.index - 1);
}

// Hide polylines for trackers other than the active playback tracker
function hideOtherTrackersPolylines(activeId){
    trackers.forEach((t) => {
        if (t.Id !== activeId && t.Polyline && !t.PolylineHiddenDueToPlayback){
            if (t.Visible){ t.Polyline.remove(); }
            t.PolylineHiddenDueToPlayback = true;
        }
    });
}

// Restore polylines hidden due to playback
function restoreHiddenPolylines(){
    trackers.forEach((t) => {
        if (t.PolylineHiddenDueToPlayback){
            if (t.Visible && t.Polyline){ t.Polyline.addTo(map); }
            delete t.PolylineHiddenDueToPlayback;
        }
    });
}

// Optional: keyboard shortcuts for stepping
document.addEventListener('keydown', (e) => {
    const panel = document.getElementById('playbackPanel');
    if (panel && panel.classList.contains('hidden')) return;
    if (e.key === 'ArrowRight') { stepForward(); }
    if (e.key === 'ArrowLeft') { stepBack(); }
});

/**
 * Fetch data from API of all trackers, and update tracker properties
 */
async function refreshTrackers(){
    const latestData = await fetchTrackers();
    latestData.forEach(newTrackerData => {
        if (trackers.has(newTrackerData.Id)){
            let tracker = trackers.get(newTrackerData.Id)
            if (tracker.Timestamp < newTrackerData.Timestamp){
                updateTrackerLocation(tracker,newTrackerData) 
            }
        }
    });
}

/**
 * Fetch data of all tracker using API 
 * @returns {Promise<Array>} - List of trackers API data
 */
async function fetchTrackers(){
    return apiGet("trackers");
}

/**
 * Small fetch helper for GET requests that adds headers and error handling
 * @param {string} path - API path under baseurl
 * @returns {Promise<any>} - Parsed JSON response
 */
async function apiGet(path){
    try{
        const res = await fetch(baseurl+path, {
            method: "GET",
            headers: {
                "X-API-Key": `${apiKey}`,
                "Content-Type": "application/json"
            }
        });
        if (!res.ok){
            const text = await res.text();
            throw new Error(`GET ${path} failed: ${res.status} ${text}`);
        }
        return res.json();
    } catch (err){
        console.error(err);
        throw err;
    }
}

/*
 * Initialize the map, sidebar and tracker data
*/ 
async function initializeMapAndTracker(){
    // Async start fetching trackers
    const fetchTrackersPromise = fetchTrackers();

    // Load map
    map = L.map('map',{fullscreenControl: false, minZoom:0, maxZoom:19, zoomControl:false})
    map.setView([55.691175204581256, 12.53654950759347], 12);
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    }).addTo(map);

    const trackersData = await fetchTrackersPromise;
    // Add trackers to map after fetching has completed, and the map is loaded
    await initiateTrackers(trackersData);
    renderTrackersList();
    // hide loading overlay
    const loading = document.getElementById("loading");
    if (loading){ loading.classList.add("hidden"); loading.setAttribute("aria-busy","false"); }

    // Load sidebar
    let sidebar = L.control.sidebar('sidebar').addTo(map);
    L.control.zoom({position: 'bottomright'}).addTo(map);
    // Fetch and redraw tracker every 10 seconds
    setInterval(refreshTrackers,10*1000);
}

/*
 * Validate that apikey is set and valid, and display according error message
*/ 
async function validateApiKey(){
    if (!apiKey){
        window.alert("Invalid URL. URL must specify an API KEY");
        return false;
    }
    try{
        await apiGet("whoami");
        return true;
    } catch (err){
        if (String(err).includes("401")){
            window.alert("API key is invalid");
        } else {
            window.alert("Failed to validate API key");
        }
        return false;
    }
}

/*
 * Initialize map and trackers, if api key is valid
*/ 
async function initializeApp(){
    const validKey = await validateApiKey();
    if (validKey){
        initializeMapAndTracker();
    }
}
initializeApp();
