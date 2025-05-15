const baseurl = "https://api.banjo.dev:8080/api/v1/";
const apiKey = new URLSearchParams(document.location.search).get("api-key");

// Leaflet map
let map;

// hashmap of trackers
const trackers = new Map();

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
    trackers.values().forEach(tracker => setTrackerVisibility(tracker,visible));
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
        } else {
            tracker.Marker.remove();
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
        zoomToTracker(t);
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
 * Store trackers in global variable, create markers for each 
 * @param {Array} data - List af tracker data from API
 */
async function initiateTrackers(data){
    const markericon = new L.Icon.Default();
    markericon.options.shadowSize = [0,0];
    data.forEach((tracker) => {
        tracker.Visible = true;
        tracker.Marker = new L.Marker({icon:markericon});
        trackers.set(tracker.Id,tracker);  
        updateTrackerLocation(tracker,tracker)
        tracker.Marker.addTo(map)
    });
}

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
    return fetch(baseurl+"trackers", {
        method: "GET",
        headers: {
            "X-API-Key": `${apiKey}`,
            "Content-Type": "application/json"
        }
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`Error. Statuscode: ${response.status}`);
            }
            return response.json();
        })
        .catch(error => console.error("Error:", error));
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
    initiateTrackers(trackersData);
    renderTrackersList();

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
    return await fetch(baseurl+"whoami", {
        method: "GET",
        headers: {
            "X-API-Key": `${apiKey}`,
            "Content-Type": "application/json"
        }
    })
        .then(response => {
            if (response.ok){
                return true
            }
            if (response.status == 401){
                window.alert("API key is invalid");
            }
            else {
                window.alert("Failed to validate API key")
            }
            return false;
        })
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
