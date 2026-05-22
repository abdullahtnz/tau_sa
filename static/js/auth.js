// auth.js - Login page logic

var currentRole = "student";

document.addEventListener("DOMContentLoaded", function () {
    var passwordInput = document.getElementById("password");
    if (passwordInput) {
        passwordInput.addEventListener("keydown", function (e) {
            if (e.key === "Enter") login();
        });
    }
});

function switchTab(role) {
    currentRole = role;

    var tabs = document.querySelectorAll(".tab");
    for (var i = 0; i < tabs.length; i++) {
        tabs[i].classList.remove("active");
        if (tabs[i].textContent.toLowerCase() === role) {
            tabs[i].classList.add("active");
        }
    }

    document.getElementById("login-title").textContent =
        role === "student" ? "Student Login" : "Teacher Login";
    document.getElementById("id-label").textContent =
        role === "student" ? "Student ID" : "Teacher ID";
    document.getElementById("user-id").placeholder =
        role === "student" ? "Enter your student ID" : "Enter your teacher ID";

    hideError();
}

function login() {
    var userId = document.getElementById("user-id").value.trim();
    var password = document.getElementById("password").value.trim();

    if (!userId || !password) {
        showError("Please fill in all fields.");
        return;
    }

    hideError();

    var endpoint =
        currentRole === "student"
            ? api("/api/login/student")
            : api("/api/login/teacher");

    console.log("Login attempt:", currentRole, userId, "->", endpoint);

    fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: userId, password: password }),
    })
        .then(function (res) {
            console.log("Login response status:", res.status);
            return res.json().then(function (data) {
                return { ok: res.ok, data: data };
            }, function () {
                return { ok: res.ok, data: null };
            });
        })
        .then(function (result) {
            console.log("Login result:", result);
            if (!result.ok) {
                var errMsg = (result.data && result.data.error) ? result.data.error : "Login failed.";
                showError(errMsg);
                return;
            }

            localStorage.setItem("token", result.data.token);
            localStorage.setItem("user_id", result.data.user_id);
            localStorage.setItem("full_name", result.data.full_name);
            localStorage.setItem("role", currentRole);

            console.log("Login success, redirecting to dashboard");

            if (currentRole === "student") {
                window.location.href = pathTo("student-dashboard.html");
            } else {
                window.location.href = pathTo("teacher-dashboard.html");
            }
        })
        .catch(function (err) {
            console.error("Login error:", err);
            showError("Network error. Please try again.");
        });
}

function showError(msg) {
    var el = document.getElementById("login-error");
    el.textContent = msg;
    el.style.display = "block";
}

function hideError() {
    document.getElementById("login-error").style.display = "none";
}
