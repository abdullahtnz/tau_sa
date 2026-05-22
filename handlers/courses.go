package handlers

import (
	"net/http"

	"tau_smart_attendance/database"
	"tau_smart_attendance/middleware"
	"tau_smart_attendance/models"
)

func GetTeacherCourses(w http.ResponseWriter, r *http.Request) {
	teacherID := r.Context().Value(middleware.UserIDKey).(int)

	rows, err := database.DB.Query(`
		SELECT c.id, c.course_code, c.course_name, COALESCE(c.department, '')
		FROM courses c
		INNER JOIN teacher_courses tc ON c.id = tc.course_id
		WHERE tc.teacher_id = $1
		ORDER BY c.course_code
	`, teacherID)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch courses"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	courses := []models.Course{}
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(&c.ID, &c.CourseCode, &c.CourseName, &c.Department); err != nil {
			continue
		}
		courses = append(courses, c)
	}

	writeJSON(w, courses)
}

func GetTeacherClassSessions(w http.ResponseWriter, r *http.Request) {
	teacherID := r.Context().Value(middleware.UserIDKey).(int)

	rows, err := database.DB.Query(`
		SELECT cs.id, cs.course_id, cs.teacher_id, cs.session_date, cs.created_at, cs.updated_at,
		       c.course_code, c.course_name
		FROM class_sessions cs
		INNER JOIN courses c ON cs.course_id = c.id
		WHERE cs.teacher_id = $1
		ORDER BY cs.session_date DESC, cs.created_at DESC
	`, teacherID)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch class sessions"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	sessions := []models.ClassSession{}
	for rows.Next() {
		var s models.ClassSession
		if err := rows.Scan(&s.ID, &s.CourseID, &s.TeacherID, &s.SessionDate, &s.CreatedAt, &s.UpdatedAt, &s.CourseCode, &s.CourseName); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}

	writeJSON(w, sessions)
}

func CreateClassSession(w http.ResponseWriter, r *http.Request) {
	teacherID := r.Context().Value(middleware.UserIDKey).(int)

	var req models.CreateClassSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var exists bool
	err := database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM class_sessions WHERE course_id=$1 AND teacher_id=$2 AND session_date=$3)",
		req.CourseID, teacherID, req.SessionDate,
	).Scan(&exists)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, `{"error":"class session already exists for this course and date"}`, http.StatusConflict)
		return
	}

	var session models.ClassSession
	err = database.DB.QueryRow(`
		INSERT INTO class_sessions (course_id, teacher_id, session_date)
		VALUES ($1, $2, $3)
		RETURNING id, course_id, teacher_id, session_date, created_at, updated_at
	`, req.CourseID, teacherID, req.SessionDate).Scan(
		&session.ID, &session.CourseID, &session.TeacherID, &session.SessionDate, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		http.Error(w, `{"error":"failed to create class session"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, session)
}

func GetClassSessionAttendance(w http.ResponseWriter, r *http.Request) {
	sessionID := chiURLParam(r, "id")

	rows, err := database.DB.Query(`
		SELECT ar.id, ar.class_session_id, ar.student_id, ar.qr_session_id,
		       ar.device_fingerprint, ar.attended_at,
		       s.full_name, s.student_id as student_no
		FROM attendance_records ar
		INNER JOIN students s ON ar.student_id = s.id
		WHERE ar.class_session_id = $1
		ORDER BY ar.attended_at ASC
	`, sessionID)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch attendance"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records := []models.AttendanceRecord{}
	for rows.Next() {
		var rec models.AttendanceRecord
		if err := rows.Scan(&rec.ID, &rec.ClassSessionID, &rec.StudentID, &rec.QRSessionID,
			&rec.DeviceFingerprint, &rec.AttendedAt, &rec.StudentName, &rec.StudentNo); err != nil {
			continue
		}
		records = append(records, rec)
	}

	writeJSON(w, records)
}
