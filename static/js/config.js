// config.js - API configuration
// Change API_BASE to your backend URL when deploying separately.
// Leave empty ("") when frontend is served from the same origin as the backend.

var API_BASE = "https://tau-sa.onrender.com";

// Example for separate deployment:
// var API_BASE = "https://your-backend-server.com";

function api(path) {
    return API_BASE + path;
}