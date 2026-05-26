package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"tau_smart_attendance/database"
	"tau_smart_attendance/middleware"
	"tau_smart_attendance/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
}

var dummyPasswordHash string

func init() {
	hash, err := bcrypt.GenerateFromPassword([]byte("tau_dummy_timing_protection_2026"), 12)
	if err != nil {
		log.Fatalf("Failed to generate dummy password hash: %v", err)
	}
	dummyPasswordHash = string(hash)
}

func StudentLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.UserID) == 0 || len(req.UserID) > 255 || len(req.Password) == 0 || len(req.Password) > 255 {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var student models.Student
	var passwordHash string
	userFound := true

	err := database.DB.QueryRow(context.Background(),
		"SELECT id, student_id, password_hash, full_name, email, department FROM students WHERE student_id = $1",
		req.UserID,
	).Scan(&student.ID, &student.StudentID, &student.PasswordHash, &student.FullName, &student.Email, &student.Department)

	if err == pgx.ErrNoRows {
		userFound = false
		passwordHash = dummyPasswordHash
	} else if err != nil {
		log.Printf("StudentLogin query error: %v", err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	} else {
		passwordHash = student.PasswordHash
	}

	compareErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))

	if !userFound || compareErr != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(student.ID, "student", student.StudentID)
	if err != nil {
		log.Printf("StudentLogin token generation error: %v", err)
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

	if len(req.UserID) == 0 || len(req.UserID) > 255 || len(req.Password) == 0 || len(req.Password) > 255 {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var teacher models.Teacher
	var passwordHash string
	userFound := true

	err := database.DB.QueryRow(context.Background(),
		"SELECT id, teacher_id, password_hash, full_name, email, department FROM teachers WHERE teacher_id = $1",
		req.UserID,
	).Scan(&teacher.ID, &teacher.TeacherID, &teacher.PasswordHash, &teacher.FullName, &teacher.Email, &teacher.Department)

	if err == pgx.ErrNoRows {
		userFound = false
		passwordHash = dummyPasswordHash
	} else if err != nil {
		log.Printf("TeacherLogin query error: %v", err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	} else {
		passwordHash = teacher.PasswordHash
	}

	compareErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))

	if !userFound || compareErr != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(teacher.ID, "teacher", teacher.TeacherID)
	if err != nil {
		log.Printf("TeacherLogin token generation error: %v", err)
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
	jti := generateJTI()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"role":     role,
		"login_id": loginID,
		"jti":      jti,
		"exp":      time.Now().Add(8 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"iss":      "tau-smart-attendance",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(middleware.JWTSecret)
}

func generateJTI() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Printf("generateJTI random error: %v", err)
		fallback := make([]byte, 16)
		copy(fallback, []byte(time.Now().String()))
		return hex.EncodeToString(fallback)
	}
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
