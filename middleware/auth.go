package middleware

import (
    "context"
    "net/http"
    "os"
    "strings"
    "github.com/golang-jwt/jwt/v5"
)

var JWTSecret []byte

func init() {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        panic("JWT_SECRET environment variable not set")
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

        userIDFloat, ok := claims["user_id"].(float64)
        if !ok {
            http.Error(w, `{"error":"invalid user id in token"}`, http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), UserIDKey, int(userIDFloat))
        ctx = context.WithValue(ctx, UserRoleKey, "student")
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Same for TeacherAuth...

func parseToken(tokenStr string) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        return JWTSecret, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if !token.Valid {
        return nil, jwt.ErrSignatureInvalid
    }
    
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, jwt.ErrSignatureInvalid
    }
    
    return claims, nil
}