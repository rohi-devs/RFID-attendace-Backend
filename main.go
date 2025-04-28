package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/cors"
)

var db *sql.DB

func main() {
	var err error

	db, err = sql.Open("pgx", "postgresql://neondb_owner:npg_aSm2XKLDbu5J@ep-nameless-grass-a1hhakn0-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/walkin/{rfid}", WalkIn).Methods("POST")
	r.HandleFunc("/walkout/{rfid}", WalkOut).Methods("POST")
	r.HandleFunc("/student/{rfid}/attendance", GetStudentAttendance).Methods("GET")
	r.HandleFunc("/dashboard/present", GetCurrentlyPresentStudents).Methods("GET")
	r.HandleFunc("/dashboard/history", GetAttendanceHistory).Methods("GET")

	// Create a CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"}, // Allow all headers
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

	// Wrap the router with the CORS handler
	handler := c.Handler(r)

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// Walk-in (register in-time)
func WalkIn(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rfid := vars["rfid"]

	var studentID int
	err := db.QueryRow("SELECT id FROM students WHERE rfid_id = $1", rfid).Scan(&studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec("INSERT INTO attendance (student_id, in_time) VALUES ($1, $2)", studentID, time.Now())
	if err != nil {
		http.Error(w, "Failed to register walk-in", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Walk-in registered successfully"))
}

// Walk-out (register out-time and calculate total time)
func WalkOut(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rfid := vars["rfid"]

	var studentID int
	err := db.QueryRow("SELECT id FROM students WHERE rfid_id = $1", rfid).Scan(&studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	var attendanceID int
	var inTime time.Time
	err = db.QueryRow("SELECT id, in_time FROM attendance WHERE student_id = $1 AND out_time IS NULL ORDER BY in_time DESC LIMIT 1", studentID).
		Scan(&attendanceID, &inTime)
	if err != nil {
		http.Error(w, "No active walk-in found for walk-out", http.StatusBadRequest)
		return
	}

	now := time.Now()
	totalDuration := now.Sub(inTime)

	_, err = db.Exec("UPDATE attendance SET out_time = $1, total_duration = $2 WHERE id = $3", now, totalDuration, attendanceID)
	if err != nil {
		http.Error(w, "Failed to register walk-out", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Walk-out registered successfully"))
}

// Get attendance history for a specific student
func GetStudentAttendance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rfid := vars["rfid"]

	var studentID int
	err := db.QueryRow("SELECT id FROM students WHERE rfid_id = $1", rfid).Scan(&studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	rows, err := db.Query("SELECT in_time, out_time, total_duration FROM attendance WHERE student_id = $1 ORDER BY in_time DESC", studentID)
	if err != nil {
		http.Error(w, "Failed to fetch attendance records", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result string
	for rows.Next() {
		var inTime, outTime sql.NullTime
		var totalDuration sql.NullString
		err = rows.Scan(&inTime, &outTime, &totalDuration)
		if err != nil {
			continue
		}
		result += fmt.Sprintf("In: %v, Out: %v, Duration: %v\n", inTime.Time.Format(time.RFC3339), outTime.Time.Format(time.RFC3339), totalDuration.String)
	}

	if result == "" {
		w.Write([]byte("No attendance records found"))
		return
	}

	w.Write([]byte(result))
}

// Dashboard: Get currently present students
func GetCurrentlyPresentStudents(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
        SELECT s.name, s.department, a.in_time
        FROM students s
        JOIN attendance a ON s.id = a.student_id
        WHERE a.out_time IS NULL
        ORDER BY a.in_time
    `)
	if err != nil {
		http.Error(w, "Failed to fetch current students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result string
	for rows.Next() {
		var name, department string
		var inTime time.Time
		err = rows.Scan(&name, &department, &inTime)
		if err != nil {
			continue
		}
		result += fmt.Sprintf("Name: %s, Department: %s, In Time: %v\n", name, department, inTime.Format(time.RFC3339))
	}

	if result == "" {
		w.Write([]byte("No students currently present"))
		return
	}

	w.Write([]byte(result))
}

// Dashboard: Get full attendance history
func GetAttendanceHistory(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
        SELECT s.name, a.in_time, a.out_time, a.total_duration
        FROM students s
        JOIN attendance a ON s.id = a.student_id
        ORDER BY a.in_time DESC
    `)
	if err != nil {
		http.Error(w, "Failed to fetch attendance history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result string
	for rows.Next() {
		var name string
		var inTime, outTime sql.NullTime
		var totalDuration sql.NullString
		err = rows.Scan(&name, &inTime, &outTime, &totalDuration)
		if err != nil {
			continue
		}
		result += fmt.Sprintf("Name: %s, In: %v, Out: %v, Duration: %v\n", name, inTime.Time.Format(time.RFC3339), outTime.Time.Format(time.RFC3339), totalDuration.String)
	}

	if result == "" {
		w.Write([]byte("No attendance history found"))
		return
	}

	w.Write([]byte(result))
}
