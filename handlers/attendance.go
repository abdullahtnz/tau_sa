package handlers

import (
	"net/http"
	"time"

	"tau_smart_attendance/database"
	"tau_smart_attendance/middleware"
	"tau_smart_attendance/models"
)

func SubmitAttendance(w http.ResponseWriter, r *http.Request) {
	studentID := r.Context().Value(middleware.UserIDKey).(int)

	var req models.AttendRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var storedToken string
	var isActive bool
	var tokenExpiresAt *time.Time
	var csID int
	err := database.DB.QueryRow(`
		SELECT current_token, is_active, token_expires_at, class_session_id
		FROM qr_sessions WHERE id = $1
	`, req.QRSessionID).Scan(&storedToken, &isActive, &tokenExpiresAt, &csID)

	if err != nil {
		http.Error(w, `{"error":"invalid QR session"}`, http.StatusBadRequest)
		return
	}

	if !isActive {
		http.Error(w, `{"error":"QR session is not active"}`, http.StatusBadRequest)
		return
	}

	if storedToken != req.Token {
		http.Error(w, `{"error":"invalid QR token"}`, http.StatusBadRequest)
		return
	}

	if tokenExpiresAt != nil && time.Now().After(tokenExpiresAt.Add(3*time.Second)) {
		http.Error(w, `{"error":"QR token expired"}`, http.StatusBadRequest)
		return
	}

	if csID != req.ClassSessionID {
		http.Error(w, `{"error":"mismatched class session"}`, http.StatusBadRequest)
		return
	}

	var alreadyAttended bool
	err = database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM attendance_records WHERE class_session_id=$1 AND student_id=$2)",
		req.ClassSessionID, studentID,
	).Scan(&alreadyAttended)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if alreadyAttended {
		http.Error(w, `{"error":"you have already attended this class session"}`, http.StatusConflict)
		return
	}

	var deviceUsed bool
	err = database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM attendance_records WHERE class_session_id=$1 AND device_fingerprint=$2)",
		req.ClassSessionID, req.DeviceFingerprint,
	).Scan(&deviceUsed)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if deviceUsed {
		http.Error(w, `{"error":"this device has already been used for attendance in this class session"}`, http.StatusConflict)
		return
	}

	_, err = database.DB.Exec(`
		INSERT INTO attendance_records (class_session_id, student_id, qr_session_id, device_fingerprint)
		VALUES ($1, $2, $3, $4)
	`, req.ClassSessionID, studentID, req.QRSessionID, req.DeviceFingerprint)

	if err != nil {
		http.Error(w, `{"error":"failed to record attendance"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "attendance recorded successfully"})
}

func GetStudentAttendance(w http.ResponseWriter, r *http.Request) {
	studentID := r.Context().Value(middleware.UserIDKey).(int)

	rows, err := database.DB.Query(`
		SELECT ar.id, ar.class_session_id, ar.student_id, ar.qr_session_id,
		       ar.device_fingerprint, ar.attended_at,
		       c.course_code, c.course_name, cs.session_date
		FROM attendance_records ar
		INNER JOIN class_sessions cs ON ar.class_session_id = cs.id
		INNER JOIN courses c ON cs.course_id = c.id
		WHERE ar.student_id = $1
		ORDER BY cs.session_date DESC, ar.attended_at DESC
	`, studentID)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch attendance"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records := []models.AttendanceRecord{}
	for rows.Next() {
		var rec models.AttendanceRecord
		if err := rows.Scan(&rec.ID, &rec.ClassSessionID, &rec.StudentID, &rec.QRSessionID,
			&rec.DeviceFingerprint, &rec.AttendedAt, &rec.CourseCode, &rec.CourseName, &rec.SessionDate); err != nil {
			continue
		}
		records = append(records, rec)
	}

	writeJSON(w, records)
}
