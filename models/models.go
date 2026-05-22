package models

import "time"

type Student struct {
	ID           int       `json:"id"`
	StudentID    string    `json:"student_id"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email,omitempty"`
	Department   string    `json:"department,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Teacher struct {
	ID           int       `json:"id"`
	TeacherID    string    `json:"teacher_id"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email,omitempty"`
	Department   string    `json:"department,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Course struct {
	ID         int       `json:"id"`
	CourseCode string    `json:"course_code"`
	CourseName string    `json:"course_name"`
	Department string    `json:"department,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type ClassSession struct {
	ID          int       `json:"id"`
	CourseID    int       `json:"course_id"`
	TeacherID   int       `json:"teacher_id"`
	SessionDate string    `json:"session_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CourseCode  string    `json:"course_code,omitempty"`
	CourseName  string    `json:"course_name,omitempty"`
}

type QRSession struct {
	ID                   int        `json:"id"`
	ClassSessionID       int        `json:"class_session_id"`
	IsActive             bool       `json:"is_active"`
	CurrentToken         string     `json:"current_token,omitempty"`
	TokenExpiresAt       *time.Time `json:"token_expires_at,omitempty"`
	NumericCode          string     `json:"numeric_code,omitempty"`
	NumericCodeExpiresAt *time.Time `json:"numeric_code_expires_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	ClosedAt             *time.Time `json:"closed_at,omitempty"`
	CourseCode           string     `json:"course_code,omitempty"`
	CourseName           string     `json:"course_name,omitempty"`
	SessionDate          string     `json:"session_date,omitempty"`
}

type AttendanceRecord struct {
	ID                int       `json:"id"`
	ClassSessionID    int       `json:"class_session_id"`
	StudentID         int       `json:"student_id"`
	QRSessionID       int       `json:"qr_session_id"`
	DeviceFingerprint string    `json:"device_fingerprint"`
	AttendedAt        time.Time `json:"attended_at"`
	StudentName       string    `json:"student_name,omitempty"`
	StudentNo         string    `json:"student_no,omitempty"`
	CourseCode        string    `json:"course_code,omitempty"`
	CourseName        string    `json:"course_name,omitempty"`
	SessionDate       string    `json:"session_date,omitempty"`
}

type LoginRequest struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

type AttendRequest struct {
	QRSessionID       int    `json:"qr_session_id"`
	ClassSessionID    int    `json:"class_session_id"`
	Token             string `json:"token,omitempty"`
	NumericCode       string `json:"numeric_code,omitempty"`
	DeviceFingerprint string `json:"device_fingerprint"`
}

type CreateClassSessionRequest struct {
	CourseID    int    `json:"course_id"`
	SessionDate string `json:"session_date"`
}
