// utils.js - Shared utility functions

function showToast(message, type) {
    var toast = document.getElementById("toast");
    toast.textContent = message;
    toast.className = "toast toast-" + type;
    toast.style.display = "block";
    clearTimeout(toast._hideTimeout);
    toast._hideTimeout = setTimeout(function () {
        toast.style.display = "none";
    }, 3000);
}

function logout() {
    localStorage.clear();
    window.location.href = pathTo("login.html");
}

function escapeHtml(text) {
    if (!text) return "";
    var div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
}

function pathTo(filename) {
    return filename;
}

function isTokenExpired() {
    var token = localStorage.getItem("token");
    if (!token) return true;
    try {
        var base64Url = token.split(".")[1];
        var base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
        var jsonPayload = decodeURIComponent(atob(base64).split("").map(function (c) {
            return "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2);
        }).join(""));
        var payload = JSON.parse(jsonPayload);
        var now = Math.floor(Date.now() / 1000);
        return payload.exp && payload.exp < now;
    } catch (e) {
        return false;
    }
}

function authFetch(url, options) {
    if (isTokenExpired()) {
        logout();
        return Promise.reject(new Error("token expired"));
    }

    var opts = options || {};
    opts.headers = opts.headers || {};
    opts.headers["Authorization"] = "Bearer " + localStorage.getItem("token");
    return fetch(url, opts).then(function (res) {
        if (res.status === 401) {
            logout();
            return Promise.reject(new Error("unauthorized"));
        }
        return res;
    });
}
