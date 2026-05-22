// teacher.js - Teacher dashboard logic

var qrTimerInterval = null;
var currentQRSession = null;

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

    document
        .getElementById("qr-modal")
        .addEventListener("click", function (e) {
            if (e.target === this && currentQRSession) {
                closeQR(currentQRSession.id);
            }
        });

    document
        .getElementById("attendance-modal")
        .addEventListener("click", function (e) {
            if (e.target === this) closeAttendanceModal();
        });

    loadCourses();
    loadSessions();
});

function loadCourses() {
    authFetch(api("/api/teacher/courses"))
        .then(function (res) {
            return res.json();
        })
        .then(function (courses) {
            var select = document.getElementById("course-select");
            select.innerHTML =
                '<option value="">-- Select a course --</option>';
            for (var i = 0; i < courses.length; i++) {
                select.innerHTML +=
                    '<option value="' +
                    courses[i].id +
                    '">' +
                    escapeHtml(courses[i].course_code) +
                    " - " +
                    escapeHtml(courses[i].course_name) +
                    "</option>";
            }
        })
        .catch(function () {
            document.getElementById("course-select").innerHTML =
                '<option value="">Failed to load courses</option>';
        });
}

function loadSessions() {
    document.getElementById("sessions-loading").style.display = "block";
    document.getElementById("sessions-empty").style.display = "none";
    document.getElementById("sessions-list").innerHTML = "";

    authFetch(api("/api/teacher/class-sessions"))
        .then(function (res) {
            return res.json();
        })
        .then(function (sessions) {
            document.getElementById("sessions-loading").style.display = "none";

            if (!sessions || sessions.length === 0) {
                document.getElementById("sessions-empty").style.display =
                    "block";
                return;
            }

            var html = "";
            for (var i = 0; i < sessions.length; i++) {
                var s = sessions[i];
                html +=
                    '<div class="session-list-item">' +
                    '<div class="session-info">' +
                    '<div class="session-name">' +
                    escapeHtml(s.course_code) +
                    " - " +
                    escapeHtml(s.course_name) +
                    "</div>" +
                    '<div class="session-date">' +
                    escapeHtml(s.session_date) +
                    "</div>" +
                    "</div>" +
                    '<div class="session-actions">' +
                    '<button class="btn btn-primary btn-sm" onclick="startQR(' +
                    s.id +
                    ",'" +
                    escapeHtml(s.course_code) +
                    "','" +
                    escapeHtml(s.course_name) +
                    "','" +
                    escapeHtml(s.session_date) +
                    '\')">Start QR</button>' +
                    '<button class="btn btn-secondary btn-sm" onclick="viewAttendance(' +
                    s.id +
                    ",'" +
                    escapeHtml(s.course_code) +
                    " - " +
                    escapeHtml(s.course_name) +
                    " | " +
                    escapeHtml(s.session_date) +
                    '\')">Attendance</button>' +
                    "</div>" +
                    "</div>";
            }
            document.getElementById("sessions-list").innerHTML = html;
        })
        .catch(function () {
            document.getElementById("sessions-loading").style.display = "none";
            document.getElementById("sessions-empty").style.display = "block";
            document.getElementById("sessions-empty").textContent =
                "Failed to load sessions.";
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
        body: JSON.stringify({
            course_id: parseInt(courseId),
            session_date: sessionDate,
        }),
    })
        .then(function (res) {
            return res.json().then(function (data) {
                return { ok: res.ok, data: data };
            });
        })
        .then(function (result) {
            if (!result.ok) {
                errorEl.textContent =
                    result.data.error || "Failed to create class session.";
                errorEl.style.display = "block";
                return;
            }

            showToast("Class session created!", "success");
            loadSessions();

            var optionText =
                document.getElementById("course-select").selectedOptions[0]
                    .text;
            var parts = optionText.split(" - ");
            startQR(result.data.id, parts[0], parts[1] || "", sessionDate);
        })
        .catch(function () {
            errorEl.textContent = "Network error. Please try again.";
            errorEl.style.display = "block";
        });
}

function startQR(classSessionId, courseCode, courseName, sessionDate) {
    authFetch(api("/api/teacher/class-sessions/" + classSessionId + "/qr"), {
        method: "POST",
    })
        .then(function (res) {
            return res.json();
        })
        .then(function (qrSession) {
            currentQRSession = qrSession;

            document.getElementById("qr-course-name").textContent =
                courseCode + " - " + courseName + " | " + sessionDate;
            document.getElementById("qr-info").textContent =
                "QR Session ID: " + qrSession.id;
            document.getElementById("qr-modal").style.display = "flex";

            document.getElementById("qr-close-btn").onclick = function () {
                closeQR(qrSession.id);
            };

            refreshQRCode(qrSession.id);
            startQRTimer(qrSession.id);
        })
        .catch(function () {
            showToast("Failed to start QR session.", "error");
        });
}

function refreshQRCode(qrSessionId) {
    authFetch(api("/api/teacher/qr-sessions/" + qrSessionId + "/token"))
        .then(function (res) {
            return res.json();
        })
        .then(function (data) {
            if (!data.is_active) {
                document.getElementById("qrcode").innerHTML =
                    '<p style="color:var(--danger);">QR session is closed.</p>';
                return;
            }

            var qrPayload = JSON.stringify({
                class_session_id: data.class_session_id,
                qr_session_id: data.qr_session_id,
                token: data.token,
            });

            document.getElementById("qrcode").innerHTML = "";
            QRCode.toCanvas(
                document.createElement("canvas"),
                qrPayload,
                { width: 280 },
                function (err, canvas) {
                    if (err) {
                        document.getElementById("qrcode").textContent =
                            "QR generation failed";
                        return;
                    }
                    document.getElementById("qrcode").appendChild(canvas);
                }
            );
        })
        .catch(function () {
            document.getElementById("qrcode").innerHTML =
                '<p style="color:var(--danger);">Connection lost.</p>';
        });
}

function startQRTimer(qrSessionId) {
    clearInterval(qrTimerInterval);
    var countdown = 5;

    document.getElementById("qr-countdown").textContent = countdown;

    qrTimerInterval = setInterval(function () {
        countdown--;
        document.getElementById("qr-countdown").textContent = countdown;

        if (countdown <= 0) {
            refreshQRCode(qrSessionId);
            countdown = 5;
        }
    }, 1000);
}

function closeQR(qrSessionId) {
    authFetch(api("/api/teacher/qr-sessions/" + qrSessionId + "/close"), {
        method: "PUT",
    })
        .then(function () {
            clearInterval(qrTimerInterval);
            document.getElementById("qr-modal").style.display = "none";
            currentQRSession = null;
            showToast("QR session closed.", "success");
        })
        .catch(function () {
            showToast("Failed to close QR session.", "error");
        });
}

function viewAttendance(classSessionId, title) {
    document.getElementById("att-course-name").textContent = title;
    document.getElementById("attendance-modal").style.display = "flex";
    document.getElementById("att-loading").style.display = "block";
    document.getElementById("att-list").innerHTML = "";
    document.getElementById("att-empty").style.display = "none";

    authFetch(
        api("/api/teacher/class-sessions/" + classSessionId + "/attendance")
    )
        .then(function (res) {
            return res.json();
        })
        .then(function (records) {
            document.getElementById("att-loading").style.display = "none";

            if (!records || records.length === 0) {
                document.getElementById("att-empty").style.display = "block";
                return;
            }

            var html =
                "<table>" +
                "<thead><tr>" +
                "<th>#</th>" +
                "<th>Student Name</th>" +
                "<th>Student ID</th>" +
                "<th>Attended At</th>" +
                "</tr></thead>" +
                "<tbody>";

            for (var i = 0; i < records.length; i++) {
                var r = records[i];
                html +=
                    "<tr>" +
                    "<td>" +
                    (i + 1) +
                    "</td>" +
                    "<td><strong>" +
                    escapeHtml(r.student_name) +
                    "</strong></td>" +
                    "<td>" +
                    escapeHtml(r.student_no) +
                    "</td>" +
                    "<td>" +
                    new Date(r.attended_at).toLocaleString() +
                    "</td>" +
                    "</tr>";
            }
            html +=
                "</tbody></table>" +
                '<div style="margin-top:12px; font-size:13px; color:var(--text-light);">' +
                "Total: " +
                records.length +
                " student(s)" +
                "</div>";

            document.getElementById("att-list").innerHTML = html;
        })
        .catch(function () {
            document.getElementById("att-loading").style.display = "none";
            document.getElementById("att-empty").style.display = "block";
            document.getElementById("att-empty").textContent =
                "Failed to load attendance.";
        });
}

function closeAttendanceModal() {
    document.getElementById("attendance-modal").style.display = "none";
}
