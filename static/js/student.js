// student.js - Student dashboard logic

var videoStream = null;
var scanning = false;
var fpPromise = null;

document.addEventListener("DOMContentLoaded", function () {
    var token = localStorage.getItem("token");
    var role = localStorage.getItem("role");

    if (!token || role !== "student") {
        window.location.href = pathTo("login.html");
        return;
    }

    document.getElementById("student-name").textContent =
        localStorage.getItem("full_name") || "Student";

    loadAttendance();
});

function loadAttendance() {
    document.getElementById("loading").style.display = "block";
    document.getElementById("empty").style.display = "none";

    authFetch(api("/api/student/attendance"))
        .then(function (res) {
            if (res.status === 403) {
                document.getElementById("loading").style.display = "none";
                document.getElementById("empty").style.display = "block";
                document.getElementById("empty").textContent = "You must be on the university network to view attendance.";
                return;
            }
            return res.json().then(function (records) {
                var tableDiv = document.getElementById("attendance-table");
                document.getElementById("loading").style.display = "none";

                if (!records || records.length === 0) {
                    document.getElementById("empty").style.display = "block";
                    return;
                }

                tableDiv.innerHTML =
                    "<table><thead><tr>" +
                    "<th>Course</th><th>Course Code</th><th>Date</th><th>Attended At</th>" +
                    "</tr></thead><tbody>" +
                    records.map(function (r) {
                        return (
                            "<tr>" +
                            "<td><strong>" + escapeHtml(r.course_name) + "</strong></td>" +
                            "<td>" + escapeHtml(r.course_code) + "</td>" +
                            "<td>" + escapeHtml(r.session_date) + "</td>" +
                            "</tr>"
                        );
                    }).join("") +
                    "</tbody></table>";
            });
        })
        .catch(function () {
            document.getElementById("loading").style.display = "none";
            document.getElementById("empty").style.display = "block";
            document.getElementById("empty").textContent = "Failed to load attendance.";
        });
}

function openScanner() {
    document.getElementById("scanner-modal").style.display = "flex";
    document.getElementById("scan-status").textContent = "Point your camera at the QR code";
    document.getElementById("scan-error").style.display = "none";

    navigator.mediaDevices
        .getUserMedia({ video: { facingMode: "environment" }, audio: false })
        .then(function (stream) {
            videoStream = stream;
            var video = document.getElementById("video");
            video.srcObject = stream;
            video.play();

            fpPromise = FingerprintJS.load();
            scanning = true;
            requestAnimationFrame(scanQRCode);
        })
        .catch(function () {
            document.getElementById("scan-error").textContent = "Camera access denied or unavailable.";
            document.getElementById("scan-error").style.display = "block";
        });
}

function stopScanner() {
    scanning = false;
    if (videoStream) {
        videoStream.getTracks().forEach(function (t) { t.stop(); });
        videoStream = null;
    }
    document.getElementById("scanner-modal").style.display = "none";
}

function scanQRCode() {
    if (!scanning) return;

    var video = document.getElementById("video");
    if (video.readyState !== video.HAVE_ENOUGH_DATA) {
        requestAnimationFrame(scanQRCode);
        return;
    }

    var canvas = document.createElement("canvas");
    var ctx = canvas.getContext("2d");
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);

    var imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    var code = jsQR(imageData.data, canvas.width, canvas.height);

    if (code) {
        scanning = false;
        stopScanner();
        processQRCode(code.data);
        return;
    }

    requestAnimationFrame(scanQRCode);
}

function processQRCode(qrData) {
    try {
        var data = JSON.parse(qrData);
        if (!data.class_session_id || !data.qr_session_id || !data.token) {
            showToast("Invalid QR code format.", "error");
            return;
        }

        submitAttendanceByQR(data.qr_session_id, data.class_session_id, data.token);
    } catch (err) {
        showToast("Invalid QR code data.", "error");
    }
}

function openCodeEntry() {
    document.getElementById("code-modal").style.display = "flex";
    document.getElementById("code-error").style.display = "none";
    document.getElementById("qr-session-id-input").value = "";
    document.getElementById("numeric-code-input").value = "";
    document.getElementById("qr-session-id-input").focus();
}

function closeCodeEntry() {
    document.getElementById("code-modal").style.display = "none";
}

function submitCode() {
    var qrSessionId = parseInt(document.getElementById("qr-session-id-input").value.trim());
    var numericCode = document.getElementById("numeric-code-input").value.trim().replace(/\s/g, "");
    var errorEl = document.getElementById("code-error");

    if (!qrSessionId || !numericCode || numericCode.length !== 6 || !/^\d{6}$/.test(numericCode)) {
        errorEl.textContent = "Please enter a valid QR Session ID and a 6-digit code.";
        errorEl.style.display = "block";
        return;
    }

    errorEl.style.display = "none";

    submitAttendanceByCode(qrSessionId, numericCode);
}

function submitAttendanceByQR(qrSessionId, classSessionId, token) {
    if (!fpPromise) fpPromise = FingerprintJS.load();
    fpPromise.then(function (fp) { return fp.get(); }).then(function (result) {
        var body = {
            qr_session_id: qrSessionId,
            class_session_id: classSessionId,
            token: token,
            device_fingerprint: result.visitorId,
        };
        return doSubmitAttendance(body);
    });
}

function submitAttendanceByCode(qrSessionId, numericCode) {
    fpPromise = FingerprintJS.load();

    fpPromise.then(function (fp) { return fp.get(); }).then(function (result) {
        var body = {
            qr_session_id: qrSessionId,
            class_session_id: 0,
            numeric_code: numericCode,
            device_fingerprint: result.visitorId,
        };
        return doSubmitAttendance(body);
    });
}

function doSubmitAttendance(body) {
    return authFetch(api("/api/student/attend"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    })
        .then(function (res) {
            return res.json().then(function (data) { return { ok: res.ok, status: res.status, data: data }; });
        })
        .then(function (result) {
            if (result.ok) {
                showToast("Attendance recorded successfully!", "success");
                loadAttendance();
                closeCodeEntry();
            } else {
                if (result.status === 403) {
                    showToast("You must be on the university network to take attendance.", "error");
                    document.getElementById("code-error").textContent = "Access denied: not on the university network.";
                } else {
                    showToast(result.data.error || "Failed to record attendance.", "error");
                    document.getElementById("code-error").textContent = result.data.error || "Failed.";
                }
                document.getElementById("code-error").style.display = "block";
            }
        })
        .catch(function (err) {
            console.error("Submit attendance error:", err);
            showToast("Network error. Please try again.", "error");
        });
}
