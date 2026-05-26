// config.js - API configuration
// Leave API_BASE empty ("") when frontend is served from the same origin as the backend.
// Set to your backend URL when deploying separately (e.g. "https://api.example.com").

var API_BASE = "";

function api(path) {
    return API_BASE + path;
}