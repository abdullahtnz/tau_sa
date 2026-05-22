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
            return res.json();
        })
        .then(function (records) {
            var tableDiv = document.getElementById("attendance-table");
            document.getElementById("loading").style.display = "none";

            if (!records || records.length === 0) {
                document.getElementById("empty").style.display = "block";
                return;
            }

            tableDiv.innerHTML =
                '<table>' +
                "<thead><tr>" +
                "<th>Course</th>" +
                "<th>Course Code</th>" +
                "<th>Date</th>" +
                "<th>Attended At</th>" +
                "</tr></thead>" +
                "<tbody>" +
                records
                    .map(function (r) {
                        return (
                            "<tr>" +
                            "<td><strong>" +
                            escapeHtml(r.course_name) +
                            "</strong></td>" +
                            "<td>" +
                            escapeHtml(r.course_code) +
                            "</td>" +
                            "<td>" +
                            escapeHtml(r.session_date) +
                            "</td>" +
                            "<td>" +
                            new Date(r.attended_at).toLocaleString() +
                            "</td>" +
                            "</tr>"
                        );
                    })
                    .join("") +
                "</tbody></table>";
        })
        .catch(function () {
            document.getElementById("loading").style.display = "none";
            document.getElementById("empty").style.display = "block";
            document.getElementById("empty").textContent =
                "Failed to load attendance.";
        });
}

function openScanner() {
    document.getElementById("scanner-modal").style.display = "flex";
    document.getElementById("scan-status").textContent =
        "Point your camera at the QR code";
    document.getElementById("scan-error").style.display = "none";

    navigator.mediaDevices
        .getUserMedia({
            video: { facingMode: "environment" },
            audio: false,
        })
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
            document.getElementById("scan-error").textContent =
                "Camera access denied or unavailable.";
            document.getElementById("scan-error").style.display = "block";
        });
}

function stopScanner() {
    scanning = false;
    if (videoStream) {
        videoStream.getTracks().forEach(function (t) {
            t.stop();
        });
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

        fpPromise.then(function (fp) {
            return fp.get();
        }).then(function (result) {
            var deviceFingerprint = result.visitorId;

            return authFetch(api("/api/student/attend"), {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    qr_session_id: data.qr_session_id,
                    class_session_id: data.class_session_id,
                    token: data.token,
                    device_fingerprint: deviceFingerprint,
                }),
            });
        }).then(function (res) {
            return res.json().then(function (respData) {
                return { ok: res.ok, data: respData };
            });
        }).then(function (result) {
            if (result.ok) {
                showToast("Attendance recorded successfully!", "success");
                loadAttendance();
            } else {
                showToast(
                    result.data.error || "Failed to record attendance.",
                    "error"
                );
            }
        });
    } catch (err) {
        showToast("Invalid QR code data.", "error");
    }
}
