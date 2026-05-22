package main

import (
	"log"
	"net/http"
	"os"

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
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/login.html", http.StatusFound)
	})

	r.Post("/api/login/student", func(w http.ResponseWriter, r *http.Request) {
		handlers.StudentLogin(w, r)
	})
	r.Post("/api/login/teacher", func(w http.ResponseWriter, r *http.Request) {
		handlers.TeacherLogin(w, r)
	})

	r.Group(func(r chi.Router) {
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

		r.Get("/api/teacher/class-sessions/{id}/attendance", handlers.GetClassSessionAttendance)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
