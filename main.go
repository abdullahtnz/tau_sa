package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"tau_smart_attendance/database"
	"tau_smart_attendance/handlers"
	"tau_smart_attendance/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	database.Connect()

	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.GlobalRateLimit)

	allowedOrigin := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGIN"))
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: allowedOrigin != "*",
		MaxAge:           300,
	}))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
			next.ServeHTTP(w, r)
		})
	})

	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/login.html", http.StatusFound)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.LoginRateLimit)

		r.Post("/api/login/student", func(w http.ResponseWriter, r *http.Request) {
			handlers.StudentLogin(w, r)
		})
		r.Post("/api/login/teacher", func(w http.ResponseWriter, r *http.Request) {
			handlers.TeacherLogin(w, r)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.IPCheck)
		r.Use(middleware.StudentAuth)

		r.Post("/api/student/attend", handlers.SubmitAttendance)
		r.Get("/api/student/attendance", handlers.GetStudentAttendance)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.TeacherAuth)

		r.Get("/api/teacher/courses", handlers.GetTeacherCourses)
		r.Get("/api/teacher/class-sessions", handlers.GetTeacherClassSessions)
		r.Post("/api/teacher/class-sessions", handlers.CreateClassSession)

		r.Post("/api/teacher/class-sessions/{id}/qr", handlers.StartQRSession)
		r.Get("/api/teacher/qr-sessions/{id}/token", handlers.GetQRToken)
		r.Put("/api/teacher/qr-sessions/{id}/close", handlers.CloseQRSession)

		r.Get("/api/teacher/qr-sessions/{id}/qr-image", handlers.GetQRImage)

		r.Get("/api/teacher/class-sessions/{id}/attendance", handlers.GetClassSessionAttendance)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on :%s", port)
	log.Fatal(server.ListenAndServe())
}
