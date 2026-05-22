package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password to hash: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if password == "" {
		fmt.Println("Password cannot be empty.")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password:", err)
		os.Exit(1)
	}

	fmt.Println("\nHashed password (copy this for your SQL INSERT):")
	fmt.Println(string(hash))

	fmt.Println("\nExample INSERT statements:")
	fmt.Println()
	fmt.Println("-- Insert a student (replace values as needed)")
	fmt.Printf(`INSERT INTO students (student_id, password_hash, full_name, email, department)
VALUES ('2024001', '%s', 'John Doe', 'john@tau.edu', 'Computer Science');`, string(hash))
	fmt.Println()
	fmt.Println()
	fmt.Println("-- Insert a teacher (replace values as needed)")
	fmt.Printf(`INSERT INTO teachers (teacher_id, password_hash, full_name, email, department)
VALUES ('T001', '%s', 'Dr. Smith', 'smith@tau.edu', 'Computer Science');`, string(hash))
	fmt.Println()
	fmt.Println()
	fmt.Println("-- Insert a course")
	fmt.Println("INSERT INTO courses (course_code, course_name, department) VALUES ('MATH110', 'Calculus I', 'Mathematics');")
	fmt.Println()
	fmt.Println("-- Assign teacher to course (replace teacher_id and course_id)")
	fmt.Println("INSERT INTO teacher_courses (teacher_id, course_id) VALUES (1, 1);")
}
