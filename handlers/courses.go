package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"tau_smart_attendance/database"
	"tau_smart_attendance/middleware"
	"tau_smart_attendance/models"
)

func GetTeacherCourses(w http.ResponseWriter, r *http.Request) {
	teacherID := r.Context().Value(middleware.UserIDKey).(int)

	rows, err := database.DB.Query(context.Background(), `
		SELECT c.id, c.course_code, c.course_name, COALESCE(c.department, '')
		FROM courses c
		INNER JOIN teacher_courses tc ON c.id = tc.course_id
		WHERE tc.teacher_id = $1
		ORDER BY c.course_code
	`, teacherID)
	if err != nil {
		log.Printf("GetTeacherCourses query error: %v", err)
		http.Error(w, `{"error":"failed to fetch courses"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	courses := []models.Course{}
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(&c.ID, &c.CourseCode, &c.CourseName, &c.Department); err != nil {
			log.Printf("GetTeacherCourses scan error: %v", err)
			continue
		}
		courses = append(courses, c)
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetTeacherCourses rows iteration error: %v", err)
	}

	writeJSON(w, courses)
}

func GetTeacherClassSessions(w http.ResponseWriter, r *http.Request) {
	teacherID := r.Context().Value(middleware.UserIDKey).(int)

	rows, err := database.DB.Query(context.Background(), `
		SELECT cs.id, cs.course_id, cs.teacher_id, cs.session_date, cs.created_at, cs.updated_at,
		       c.course_code, c.course_name
		FROM class_sessions cs
		INNER JOIN courses c ON cs.course_id = c.id
		WHERE cs.teacher_id = $1
		ORDER BY cs.session_date DESC, cs.created_at DESC
	`, teacherID)
	if err != nil {
		log.Printf("GetTeacherClassSessions query error: %v", err)
		http.Error(w, `{"error":"failed to fetch class sessions"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	sessions := []models.ClassSession{}
	for rows.Next() {
		var s models.ClassSession
		var sessionDate time.Time
		if err := rows.Scan(&s.ID, &s.CourseID, &s.TeacherID, &sessionDate, &s.CreatedAt, &s.UpdatedAt, &s.CourseCode, &s.CourseName); err != nil {
			log.Printf("GetTeacherClassSessions scan error: %v", err)
			continue
		}
		s.SessionDate = sessionDate.Format("2006-01-02")
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetTeacherClassSessions rows iteration error: %v", err)
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

	if req.CourseID <= 0 {
		http.Error(w, `{"error":"invalid course id"}`, http.StatusBadRequest)
		return
	}
	if req.SessionDate == "" {
		http.Error(w, `{"error":"session date is required"}`, http.StatusBadRequest)
		return
	}

	var exists bool
	err := database.DB.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM class_sessions WHERE course_id=$1 AND teacher_id=$2 AND session_date=$3)",
		req.CourseID, teacherID, req.SessionDate,
	).Scan(&exists)
	if err != nil {
		log.Printf("CreateClassSession exists check error: %v", err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, `{"error":"class session already exists for this course and date"}`, http.StatusConflict)
		return
	}

	var session models.ClassSession
	var sessionDate time.Time
	err = database.DB.QueryRow(context.Background(), `
		INSERT INTO class_sessions (course_id, teacher_id, session_date)
		VALUES ($1, $2, $3)
		RETURNING id, course_id, teacher_id, session_date, created_at, updated_at
	`, req.CourseID, teacherID, req.SessionDate).Scan(
		&session.ID, &session.CourseID, &session.TeacherID, &sessionDate, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		log.Printf("CreateClassSession insert error: %v", err)
		http.Error(w, `{"error":"failed to create class session"}`, http.StatusInternalServerError)
		return
	}
	session.SessionDate = sessionDate.Format("2006-01-02")

	writeJSON(w, session)
}

func GetClassSessionAttendance(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chiURLParam(r, "id")
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil || sessionID <= 0 {
		http.Error(w, `{"error":"invalid session id"}`, http.StatusBadRequest)
		return
	}

	rows, err := database.DB.Query(context.Background(), `
		SELECT ar.id, ar.class_session_id, ar.student_id, ar.qr_session_id,
		       ar.device_fingerprint, ar.attended_at,
		       s.full_name, s.student_id as student_no
		FROM attendance_records ar
		INNER JOIN students s ON ar.student_id = s.id
		WHERE ar.class_session_id = $1
		ORDER BY ar.attended_at ASC
	`, sessionID)
	if err != nil {
		log.Printf("GetClassSessionAttendance query error: %v", err)
		http.Error(w, `{"error":"failed to fetch attendance"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records := []models.AttendanceRecord{}
	for rows.Next() {
		var rec models.AttendanceRecord
		if err := rows.Scan(&rec.ID, &rec.ClassSessionID, &rec.StudentID, &rec.QRSessionID,
			&rec.DeviceFingerprint, &rec.AttendedAt, &rec.StudentName, &rec.StudentNo); err != nil {
			log.Printf("GetClassSessionAttendance scan error: %v", err)
			continue
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetClassSessionAttendance rows iteration error: %v", err)
	}

	writeJSON(w, records)
}
