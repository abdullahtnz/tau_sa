// teacher.js - Teacher dashboard logic

var qrTimerInterval = null;
var codeTimerInterval = null;
var currentQRSession = null;
var currentClassSessionId = null;

document.addEventListener("DOMContentLoaded", function () {
    var token = localStorage.getItem("token");
    var role = localStorage.getItem("role");

    if (!token || role !== "teacher") {
        window.location.href = pathTo("login.html");
        return;
    }

    document.getElementById("teacher-name").textContent =
        localStorage.getItem("full_name") || "Teacher";
    document.getElementById("session-date").value = new Date()
        .toISOString()
        .split("T")[0];

    document.getElementById("qr-modal").addEventListener("click", function (e) {
        if (e.target === this && currentQRSession) {
            closeQR(currentQRSession.id);
        }
    });

    document.getElementById("attendance-modal").addEventListener("click", function (e) {
        if (e.target === this) closeAttendanceModal();
    });

    loadCourses();
    loadSessions();
});

var coursesLoading = false;

function loadCourses() {
    if (coursesLoading) return;
    coursesLoading = true;

    var select = document.getElementById("course-select");
    select.innerHTML = '<option value="">Loading courses...</option>';

    authFetch(api("/api/teacher/courses"))
        .then(function (res) { return res.json(); })
        .then(function (courses) {
            var html = '<option value="">-- Select a course --</option>';
            for (var i = 0; i < courses.length; i++) {
                html +=
                    '<option value="' + courses[i].id + '">' +
                    escapeHtml(courses[i].course_code) + " - " +
                    escapeHtml(courses[i].course_name) + "</option>";
            }
            select.innerHTML = html;
        })
        .catch(function () {
            select.innerHTML = '<option value="">Failed to load courses</option>';
        })
        .finally(function () {
            coursesLoading = false;
        });
}

var sessionsLoading = false;

function loadSessions() {
    if (sessionsLoading) return;
    sessionsLoading = true;

    document.getElementById("sessions-loading").style.display = "block";
    document.getElementById("sessions-empty").style.display = "none";

    authFetch(api("/api/teacher/class-sessions"))
        .then(function (res) { return res.json(); })
        .then(function (sessions) {
            document.getElementById("sessions-loading").style.display = "none";

            if (!sessions || sessions.length === 0) {
                document.getElementById("sessions-empty").textContent = "No class sessions yet. Create one above.";
                document.getElementById("sessions-empty").style.display = "block";
                document.getElementById("sessions-list").innerHTML = "";
                return;
            }

            var html = "";
            for (var i = 0; i < sessions.length; i++) {
                var s = sessions[i];
                html +=
                    '<div class="session-list-item">' +
                    '<div class="session-info">' +
                    '<div class="session-name">' + escapeHtml(s.course_code) + " - " + escapeHtml(s.course_name) + "</div>" +
                    '<div class="session-date">' + escapeHtml(s.session_date) + "</div>" +
                    "</div>" +
                    '<div class="session-actions">' +
                    '<button class="btn btn-primary btn-sm" onclick="startQR(' + s.id + ",'" + escapeHtml(s.course_code) + "','" + escapeHtml(s.course_name) + "','" + escapeHtml(s.session_date) + "')\">Start QR</button>" +
                    '<button class="btn btn-secondary btn-sm" onclick="viewAttendance(' + s.id + ",'" + escapeHtml(s.course_code) + " - " + escapeHtml(s.course_name) + " | " + escapeHtml(s.session_date) + "')\">Attendance</button>" +
                    "</div></div>";
            }
            document.getElementById("sessions-list").innerHTML = html;
        })
        .catch(function () {
            document.getElementById("sessions-loading").style.display = "none";
            document.getElementById("sessions-empty").textContent = "Failed to load sessions.";
            document.getElementById("sessions-empty").style.display = "block";
        })
        .finally(function () {
            sessionsLoading = false;
        });
}

function createClassSession() {
    var courseId = document.getElementById("course-select").value;
    var sessionDate = document.getElementById("session-date").value;
    var errorEl = document.getElementById("create-error");

    if (!courseId || !sessionDate) {
        errorEl.textContent = "Please select a course and date.";
        errorEl.style.display = "block";
        return;
    }

    errorEl.style.display = "none";

    authFetch(api("/api/teacher/class-sessions"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ course_id: parseInt(courseId), session_date: sessionDate }),
    })
        .then(function (res) {
            return res.json().then(function (data) { return { ok: res.ok, data: data }; });
        })
        .then(function (result) {
            if (!result.ok) {
                errorEl.textContent = result.data.error || "Failed to create class session.";
                errorEl.style.display = "block";
                return;
            }

            showToast("Class session created!", "success");
            loadSessions();

            var optionText = document.getElementById("course-select").selectedOptions[0].text;
            var parts = optionText.split(" - ");
            startQR(result.data.id, parts[0], parts[1] || "", sessionDate);
        })
        .catch(function () {
            errorEl.textContent = "Network error. Please try again.";
            errorEl.style.display = "block";
        });
}

function startQR(classSessionId, courseCode, courseName, sessionDate) {
    currentClassSessionId = classSessionId;

    authFetch(api("/api/teacher/class-sessions/" + classSessionId + "/qr"), {
        method: "POST",
    })
        .then(function (res) { return res.json(); })
        .then(function (qrSession) {
            currentQRSession = qrSession;

            console.log("QR session started:", qrSession);

            document.getElementById("qr-course-name").textContent =
                courseCode + " - " + courseName + " | " + sessionDate;
            document.getElementById("qr-info").textContent = "QR Session ID: " + qrSession.id;
            document.getElementById("qr-modal").style.display = "flex";

            document.getElementById("qr-close-btn").onclick = function () {
                closeQR(qrSession.id);
            };

            refreshQRImage(qrSession.id);
            refreshNumericCode();
            startQRTimers(qrSession.id);
        })
        .catch(function (err) {
            console.error("Failed to start QR session:", err);
            showToast("Failed to start QR session.", "error");
        });
}

function refreshQRImage(qrSessionId) {
    var img = document.getElementById("qrcode-img");

    img.onerror = function () {
        console.error("QR image failed to load");
        document.getElementById("qrcode-img").style.display = "none";
    };

    img.onload = function () {
        console.log("QR image loaded successfully");
        document.getElementById("qrcode-img").style.display = "block";
    };

    var url = api("/api/teacher/qr-sessions/" + qrSessionId + "/qr-image") +
        "?class_session_id=" + currentClassSessionId + "&t=" + Date.now();

    console.log("Loading QR image:", url);
    img.src = url;
}

function refreshNumericCode() {
    if (!currentQRSession) return;

    authFetch(api("/api/teacher/qr-sessions/" + currentQRSession.id + "/token"))
        .then(function (res) { return res.json(); })
        .then(function (data) {
            console.log("Numeric code refresh:", data);

            if (!data.is_active) {
                document.getElementById("numeric-code-display").textContent = "CLOSED";
                return;
            }

            if (data.numeric_code) {
                document.getElementById("numeric-code-display").textContent = data.numeric_code;
            }
        })
        .catch(function () {
            document.getElementById("numeric-code-display").textContent = "ERROR";
        });
}

function startQRTimers(qrSessionId) {
    clearInterval(qrTimerInterval);
    clearInterval(codeTimerInterval);

    var qrCountdown = 5;
    document.getElementById("qr-countdown").textContent = qrCountdown;

    qrTimerInterval = setInterval(function () {
        qrCountdown--;
        document.getElementById("qr-countdown").textContent = qrCountdown;

        if (qrCountdown <= 0) {
            refreshQRImage(qrSessionId);
            qrCountdown = 5;
        }
    }, 1000);

    var codeCountdown = 3;
    document.getElementById("code-countdown").textContent = codeCountdown;

    codeTimerInterval = setInterval(function () {
        codeCountdown--;
        document.getElementById("code-countdown").textContent = codeCountdown;

        if (codeCountdown <= 0) {
            refreshNumericCode();
            codeCountdown = 3;
        }
    }, 1000);
}

function closeQR(qrSessionId) {
    clearInterval(qrTimerInterval);
    qrTimerInterval = null;
    clearInterval(codeTimerInterval);
    codeTimerInterval = null;
    document.getElementById("qr-modal").style.display = "none";
    currentQRSession = null;

    authFetch(api("/api/teacher/qr-sessions/" + qrSessionId + "/close"), {
        method: "PUT",
    })
        .then(function () {
            showToast("QR session closed.", "success");
            loadSessions();
            loadCourses();
        })
        .catch(function () {
            showToast("Failed to close QR session on server.", "error");
        });
}

function viewAttendance(classSessionId, title) {
    document.getElementById("att-course-name").textContent = title;
    document.getElementById("attendance-modal").style.display = "flex";
    document.getElementById("att-loading").style.display = "block";
    document.getElementById("att-list").innerHTML = "";
    document.getElementById("att-empty").style.display = "none";

    authFetch(api("/api/teacher/class-sessions/" + classSessionId + "/attendance"))
        .then(function (res) { return res.json(); })
        .then(function (records) {
            document.getElementById("att-loading").style.display = "none";

            if (!records || records.length === 0) {
                document.getElementById("att-empty").style.display = "block";
                return;
            }

            var html =
                "<table><thead><tr>" +
                "<th>#</th><th>Student Name</th><th>Student ID</th><th>Attended At</th>" +
                "</tr></thead><tbody>";

            for (var i = 0; i < records.length; i++) {
                var r = records[i];
                html +=
                    "<tr>" +
                    "<td>" + (i + 1) + "</td>" +
                    "<td><strong>" + escapeHtml(r.student_name) + "</strong></td>" +
                    "<td>" + escapeHtml(r.student_no) + "</td>" +
                    "<td>" + new Date(r.attended_at).toLocaleString() + "</td>" +
                    "</tr>";
            }
            html += "</tbody></table>" +
                '<div style="margin-top:12px; font-size:13px; color:var(--text-light);">Total: ' + records.length + " student(s)</div>";

            document.getElementById("att-list").innerHTML = html;
        })
        .catch(function () {
            document.getElementById("att-loading").style.display = "none";
            document.getElementById("att-empty").style.display = "block";
            document.getElementById("att-empty").textContent = "Failed to load attendance.";
        });
}

function closeAttendanceModal() {
    document.getElementById("attendance-modal").style.display = "none";
}
