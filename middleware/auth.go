package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JWTSecret []byte

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "role"
	JTIKey      contextKey = "jti"
)

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("WARNING: JWT_SECRET not set in environment. Using fallback secret. This is INSECURE for production.")
		secret = "tau-smart-attendance-secret-key-2026"
	}
	if len(secret) < 32 {
		log.Printf("WARNING: JWT_SECRET is too short (%d chars). Use at least 32 characters.", len(secret))
	}
	JWTSecret = []byte(secret)
}

func StudentAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr == "" {
			http.Error(w, `{"error":"missing authorization token"}`, http.StatusUnauthorized)
			return
		}

		claims, err := parseToken(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		role, _ := claims["role"].(string)
		if role != "student" {
			http.Error(w, `{"error":"student access required"}`, http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, int(claims["user_id"].(float64)))
		ctx = context.WithValue(ctx, UserRoleKey, "student")
		if jti, ok := claims["jti"].(string); ok {
			ctx = context.WithValue(ctx, JTIKey, jti)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TeacherAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr == "" {
			http.Error(w, `{"error":"missing authorization token"}`, http.StatusUnauthorized)
			return
		}

		claims, err := parseToken(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		role, _ := claims["role"].(string)
		if role != "teacher" {
			http.Error(w, `{"error":"teacher access required"}`, http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, int(claims["user_id"].(float64)))
		ctx = context.WithValue(ctx, UserRoleKey, "teacher")
		if jti, ok := claims["jti"].(string); ok {
			ctx = context.WithValue(ctx, JTIKey, jti)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:7]) == "BEARER " {
		return bearer[7:]
	}
	return ""
}

func parseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method type: %v", t.Header["alg"])
		}
		if t.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", t.Method.Alg())
		}
		return JWTSecret, nil
	}, jwt.WithLeeway(30*time.Second))
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}
	if err := validateClaims(claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func validateClaims(claims jwt.MapClaims) error {
	if _, ok := claims["user_id"]; !ok {
		return fmt.Errorf("missing user_id claim")
	}
	if _, ok := claims["role"]; !ok {
		return fmt.Errorf("missing role claim")
	}
	if _, ok := claims["exp"]; !ok {
		return fmt.Errorf("missing exp claim")
	}
	return nil
}
