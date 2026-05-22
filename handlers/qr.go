package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"tau_smart_attendance/database"
	"tau_smart_attendance/models"

	"github.com/go-chi/chi/v5"
)

const tokenValiditySeconds = 5

func StartQRSession(w http.ResponseWriter, r *http.Request) {
	classSessionID := chiURLParam(r, "id")

	token := generateToken()
	expiresAt := time.Now().Add(time.Duration(tokenValiditySeconds) * time.Second)

	var qrSession models.QRSession
	err := database.DB.QueryRow(`
		INSERT INTO qr_sessions (class_session_id, is_active, current_token, token_expires_at)
		VALUES ($1, true, $2, $3)
		RETURNING id, class_session_id, is_active, current_token, token_expires_at, created_at
	`, classSessionID, token, expiresAt).Scan(
		&qrSession.ID, &qrSession.ClassSessionID, &qrSession.IsActive,
		&qrSession.CurrentToken, &qrSession.TokenExpiresAt, &qrSession.CreatedAt,
	)
	if err != nil {
		http.Error(w, `{"error":"failed to start QR session"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, qrSession)
}

func CloseQRSession(w http.ResponseWriter, r *http.Request) {
	qrSessionID := chiURLParam(r, "id")

	result, err := database.DB.Exec(`
		UPDATE qr_sessions SET is_active = false, closed_at = NOW()
		WHERE id = $1 AND is_active = true
	`, qrSessionID)
	if err != nil {
		http.Error(w, `{"error":"failed to close QR session"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error":"QR session not found or already closed"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]string{"message": "QR session closed"})
}

func GetQRToken(w http.ResponseWriter, r *http.Request) {
	qrSessionID := chiURLParam(r, "id")

	var qrSession models.QRSession
	var closedAt sql.NullTime
	err := database.DB.QueryRow(`
		SELECT id, class_session_id, is_active, current_token, token_expires_at, closed_at
		FROM qr_sessions WHERE id = $1
	`, qrSessionID).Scan(
		&qrSession.ID, &qrSession.ClassSessionID, &qrSession.IsActive,
		&qrSession.CurrentToken, &qrSession.TokenExpiresAt, &closedAt,
	)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"QR session not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	if !qrSession.IsActive {
		writeJSON(w, map[string]interface{}{
			"class_session_id": qrSession.ClassSessionID,
			"qr_session_id":    qrSession.ID,
			"is_active":        false,
			"token":            "",
		})
		return
	}

	if closedAt.Valid {
		qrSession.ClosedAt = &closedAt.Time
	}

	now := time.Now()
	if qrSession.TokenExpiresAt != nil && now.After(*qrSession.TokenExpiresAt) {
		token := generateToken()
		expiresAt := now.Add(time.Duration(tokenValiditySeconds) * time.Second)

		_, err := database.DB.Exec(`
			UPDATE qr_sessions SET current_token = $1, token_expires_at = $2 WHERE id = $3
		`, token, expiresAt, qrSessionID)
		if err != nil {
			http.Error(w, `{"error":"failed to refresh token"}`, http.StatusInternalServerError)
			return
		}

		qrSession.CurrentToken = token
		qrSession.TokenExpiresAt = &expiresAt
	}

	writeJSON(w, map[string]interface{}{
		"class_session_id": qrSession.ClassSessionID,
		"qr_session_id":    qrSession.ID,
		"is_active":        qrSession.IsActive,
		"token":            qrSession.CurrentToken,
	})
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
