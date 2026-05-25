package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"tau_smart_attendance/database"
	"tau_smart_attendance/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	qrcode "github.com/skip2/go-qrcode"
)

const tokenValiditySeconds = 10
const numericCodeValiditySeconds = 7

func StartQRSession(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chiURLParam(r, "id")
	classSessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid class session id"}`, http.StatusBadRequest)
		return
	}

	token := generateToken()
	tokenExpires := time.Now().Add(time.Duration(tokenValiditySeconds) * time.Second)

	numericCode := generateNumericCode()
	numericExpires := time.Now().Add(time.Duration(numericCodeValiditySeconds) * time.Second)

	var qrSession models.QRSession
	err = database.DB.QueryRow(context.Background(), `
		INSERT INTO qr_sessions (class_session_id, is_active, current_token, token_expires_at, numeric_code, numeric_code_expires_at)
		VALUES ($1, true, $2, $3, $4, $5)
		RETURNING id, class_session_id, is_active, current_token, token_expires_at, numeric_code, numeric_code_expires_at, created_at
	`, classSessionID, token, tokenExpires, numericCode, numericExpires).Scan(
		&qrSession.ID, &qrSession.ClassSessionID, &qrSession.IsActive,
		&qrSession.CurrentToken, &qrSession.TokenExpiresAt,
		&qrSession.NumericCode, &qrSession.NumericCodeExpiresAt,
		&qrSession.CreatedAt,
	)
	if err != nil {
		log.Printf("StartQRSession error: %v", err)
		http.Error(w, `{"error":"failed to start QR session"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, qrSession)
}

func CloseQRSession(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chiURLParam(r, "id")
	qrSessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid qr session id"}`, http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(context.Background(), `
		UPDATE qr_sessions SET is_active = false, closed_at = NOW()
		WHERE id = $1 AND is_active = true
	`, qrSessionID)
	if err != nil {
		log.Printf("CloseQRSession error: %v", err)
		http.Error(w, `{"error":"failed to close QR session"}`, http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, `{"error":"QR session not found or already closed"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]string{"message": "QR session closed"})
}

func GetQRToken(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chiURLParam(r, "id")
	qrSessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid qr session id"}`, http.StatusBadRequest)
		return
	}

	var qrSession models.QRSession
	var closedAt *time.Time
	err = database.DB.QueryRow(context.Background(), `
		SELECT id, class_session_id, is_active, current_token, token_expires_at, numeric_code, numeric_code_expires_at, closed_at
		FROM qr_sessions WHERE id = $1
	`, qrSessionID).Scan(
		&qrSession.ID, &qrSession.ClassSessionID, &qrSession.IsActive,
		&qrSession.CurrentToken, &qrSession.TokenExpiresAt,
		&qrSession.NumericCode, &qrSession.NumericCodeExpiresAt,
		&closedAt,
	)
	if err == pgx.ErrNoRows {
		http.Error(w, `{"error":"QR session not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("GetQRToken error: %v", err)
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	if !qrSession.IsActive {
		writeJSON(w, map[string]interface{}{
			"class_session_id": qrSession.ClassSessionID,
			"qr_session_id":    qrSession.ID,
			"is_active":        false,
			"token":            "",
			"numeric_code":     "",
		})
		return
	}

	qrSession.ClosedAt = closedAt

	now := time.Now()
	updateStmt := ""
	updateArgs := []interface{}{}

	if qrSession.TokenExpiresAt != nil && now.After(*qrSession.TokenExpiresAt) {
		token := generateToken()
		tokenExpires := now.Add(time.Duration(tokenValiditySeconds) * time.Second)
		qrSession.CurrentToken = token
		qrSession.TokenExpiresAt = &tokenExpires
		updateStmt += "current_token = $" + fmt.Sprint(len(updateArgs)+1) + ", token_expires_at = $" + fmt.Sprint(len(updateArgs)+2)
		updateArgs = append(updateArgs, token, tokenExpires)
	}

	if qrSession.NumericCodeExpiresAt != nil && now.After(*qrSession.NumericCodeExpiresAt) {
		numericCode := generateNumericCode()
		numericExpires := now.Add(time.Duration(numericCodeValiditySeconds) * time.Second)
		qrSession.NumericCode = numericCode
		qrSession.NumericCodeExpiresAt = &numericExpires
		if updateStmt != "" {
			updateStmt += ", "
		}
		updateStmt += "numeric_code = $" + fmt.Sprint(len(updateArgs)+1) + ", numeric_code_expires_at = $" + fmt.Sprint(len(updateArgs)+2)
		updateArgs = append(updateArgs, numericCode, numericExpires)
	}

	if updateStmt != "" {
		_, err := database.DB.Exec(context.Background(), "UPDATE qr_sessions SET "+updateStmt+" WHERE id = $"+fmt.Sprint(len(updateArgs)+1),
			append(updateArgs, qrSessionID)...)
		if err != nil {
			log.Printf("GetQRToken update error: %v", err)
			http.Error(w, `{"error":"failed to refresh token"}`, http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, map[string]interface{}{
		"class_session_id": qrSession.ClassSessionID,
		"qr_session_id":    qrSession.ID,
		"is_active":        qrSession.IsActive,
		"token":            qrSession.CurrentToken,
		"numeric_code":     qrSession.NumericCode,
	})
}

func GetQRImage(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chiURLParam(r, "id")
	qrSessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusNotFound)
		return
	}

	var currentToken string
	var isActive bool
	err = database.DB.QueryRow(context.Background(),
		"SELECT current_token, is_active FROM qr_sessions WHERE id = $1",
		qrSessionID,
	).Scan(&currentToken, &isActive)

	if err != nil || !isActive {
		http.Error(w, "session not active", http.StatusNotFound)
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"class_session_id": chiURLParam(r, "class_session_id"),
		"qr_session_id":    qrSessionID,
		"token":            currentToken,
	})

	if r.URL.Query().Get("class_session_id") != "" {
		payload, _ = json.Marshal(map[string]interface{}{
			"class_session_id": r.URL.Query().Get("class_session_id"),
			"qr_session_id":    qrSessionID,
			"token":            currentToken,
		})
	}

	pngData, err := qrcode.Encode(string(payload), qrcode.Medium, 280)
	if err != nil {
		http.Error(w, "QR generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(pngData)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateNumericCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "000000"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
