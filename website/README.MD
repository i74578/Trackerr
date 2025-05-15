# Website
This directory contains the code for the front-end website.
The website is mainly meant to be used for debugging and testing, since it is expected that the API will be integrated into another preexisting solution. The website is able to display all trackers associated to an API key on a [Leaflet](https://github.com/Leaflet/Leaflet) map. It also uses a modified version of [sidebar-v2](https://github.com/Turbo87/sidebar-v2) to display a sidebar with a list of trackers associated. 

 ## Preview
<p align="center">
  <img src="../docs/screenshots/website.png" width="70%">
</p>

## Prerequisites
- **Browser-only**
  - Only a modern web browser
- **NPX-powered local server**
  - Node.js (≥8)
- **NGINX deployment**
  - NGINX installed and running

## Installation
Clone the repository
```
git clone https://github.com/i74578/Trackerr
cd Trackerr/website
```

## Configuration
Before using the website, the baseurl variable must be set to the API server used.
```
nano nano js/script.js
```

## Setup
### Option A: open directly
Open the website/index.html file 
### Option B:
Run it using npx for a simple http server
```
npx http-server
```
### Option C(Recommended):
Install and configure a nginx web server with SSL certificates. Then move the files inside website/ into the nginx server root(/var/www/...) directory

## Usage
The website tightly copuled to the REST API. It does not work without providing an API key. The API key is provided by appending **?api-key=<API_KEY>** to the URL and substitude **<API-KEY>** with the actual api-key, which resides in the users table in the backend database. The default key is **AAAAAA**.\
The website has a sidebar on the left, which shows a list of all trackers associated and their properties. Each tracker entry, has a **Show on map** checkbox making it possible to only show a subset of the trackers.\
The location of all trackers on the map is refreshed every 10 seconds.

