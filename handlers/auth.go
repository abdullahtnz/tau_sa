package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"tau_smart_attendance/database"
	"tau_smart_attendance/middleware"
	"tau_smart_attendance/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
}

func StudentLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var student models.Student
	err := database.DB.QueryRow(
		"SELECT id, student_id, password_hash, full_name, email, department FROM students WHERE student_id = $1",
		req.UserID,
	).Scan(&student.ID, &student.StudentID, &student.PasswordHash, &student.FullName, &student.Email, &student.Department)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(student.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(student.ID, "student", student.StudentID)
	if err != nil {
		http.Error(w, `{"error":"token generation failed"}`, http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{
		Token:    token,
		UserID:   student.StudentID,
		FullName: student.FullName,
	}
	writeJSON(w, resp)
}

func TeacherLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var teacher models.Teacher
	err := database.DB.QueryRow(
		"SELECT id, teacher_id, password_hash, full_name, email, department FROM teachers WHERE teacher_id = $1",
		req.UserID,
	).Scan(&teacher.ID, &teacher.TeacherID, &teacher.PasswordHash, &teacher.FullName, &teacher.Email, &teacher.Department)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(teacher.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(teacher.ID, "teacher", teacher.TeacherID)
	if err != nil {
		http.Error(w, `{"error":"token generation failed"}`, http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{
		Token:    token,
		UserID:   teacher.TeacherID,
		FullName: teacher.FullName,
	}
	writeJSON(w, resp)
}

func generateJWT(userID int, role, loginID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"role":     role,
		"login_id": loginID,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(middleware.JWTSecret)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
