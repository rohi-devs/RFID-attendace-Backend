package Handler

import (
	"database/sql"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"rfidattendance/MiddleWare"
	"rfidattendance/Models"
	"time"
)

// @Summary Register student walk-in
// @Description Records when a student enters using their RFID card
// @Tags attendance
// @Accept json
// @Produce json
// @Param rfid path string true "Student RFID number"
// @Success 200 {string} string "Walk-in registered successfully"
// @Failure 404 {string} string "Student not found"
// @Failure 500 {string} string "Failed to register walk-in"
// @Router /walkin/{rfid} [post]
func WalkIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	rfid := vars["rfid"]
	db, ok := MiddleWare.GetDB(r.Context())
	if !ok {
		http.Error(w, "DB not available", http.StatusInternalServerError)
		return
	}
	var studentID int
	err := db.QueryRow("SELECT id FROM students WHERE rfid_id = $1", rfid).Scan(&studentID)
	if err != nil {
		json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "Student not found",
		})
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_, err = db.Exec("INSERT INTO attendance (student_id, in_time) VALUES ($1, $2)", studentID, time.Now())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "Failed to register walk-in",
		})
		return
	}

	json.NewEncoder(w).Encode(Models.Response{
		Status:  "success",
		Message: "Walk-in registered successfully",
	})
}

// @Summary Register student walk-out
// @Description Records when a student exits using their RFID card
// @Tags attendance
// @Accept json
// @Produce json
// @Param rfid path string true "Student RFID number"
// @Success 200 {string} string "Walk-out registered successfully"
// @Failure 404 {string} string "Student not found"
// @Failure 400 {string} string "No active walk-in found for walk-out"
// @Failure 500 {string} string "Failed to register walk-out"
// @Router /walkout/{rfid} [post]
func WalkOut(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	rfid := vars["rfid"]
	db, ok := MiddleWare.GetDB(r.Context())
	if !ok {
		http.Error(w, "DB not available", http.StatusInternalServerError)
		return
	}
	var studentID int
	err := db.QueryRow("SELECT id FROM students WHERE rfid_id = $1", rfid).Scan(&studentID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "Student not found",
		})
		return
	}

	var attendanceID int
	var inTime time.Time
	err = db.QueryRow("SELECT id, in_time FROM attendance WHERE student_id = $1 AND out_time IS NULL ORDER BY in_time DESC LIMIT 1", studentID).
		Scan(&attendanceID, &inTime)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "No active walk-in found for walk-out",
		})
		return
	}

	now := time.Now()
	totalDuration := now.Sub(inTime)

	_, err = db.Exec("UPDATE attendance SET out_time = $1, total_duration = $2 WHERE id = $3", now, totalDuration, attendanceID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "Failed to register walk-out",
		})
		return
	}

	json.NewEncoder(w).Encode(Models.Response{
		Status:  "success",
		Message: "Walk-out registered successfully",
	})
}

// @Summary Get student attendance history
// @Description Retrieves attendance history for a specific student
// @Tags students
// @Accept json
// @Produce json
// @Param rfid path string true "Student RFID number"
// @Success 200 {object} Response{data=[]AttendanceRecord} "Attendance records retrieved successfully"
// @Failure 404 {object} Response "Student not found"
// @Failure 500 {object} Response "Failed to fetch attendance records"
// @Router /student/{rfid}/attendance [get]
func GetStudentAttendance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	rfid := vars["rfid"]
	db, ok := MiddleWare.GetDB(r.Context())
	if !ok {
		http.Error(w, "DB not available", http.StatusInternalServerError)
		return
	}
	var studentID int
	err := db.QueryRow("SELECT id FROM students WHERE rfid_id = $1", rfid).Scan(&studentID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		err := json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "Student not found",
		})
		if err != nil {
			return
		}
		return
	}

	// Update the query to include rfid_id
	rows, err := db.Query(`
        SELECT s.rfid_id, a.in_time, a.out_time, a.total_duration 
        FROM attendance a
        JOIN students s ON s.id = a.student_id 
        WHERE s.id = $1 
        ORDER BY a.in_time DESC`, studentID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "Failed to fetch attendance records",
		})
		return
	}
	defer rows.Close()

	var records []Models.AttendanceRecord
	for rows.Next() {
		var rfid string
		var inTime, outTime sql.NullTime
		var totalDuration sql.NullString
		err = rows.Scan(&rfid, &inTime, &outTime, &totalDuration)
		if err != nil {
			continue
		}
		records = append(records, Models.AttendanceRecord{
			RFID:     rfid,
			InTime:   inTime.Time.Format(time.RFC3339),
			OutTime:  outTime.Time.Format(time.RFC3339),
			Duration: totalDuration.String,
		})
	}

	json.NewEncoder(w).Encode(Models.Response{
		Status:  "success",
		Message: "Attendance records retrieved successfully",
		Data:    records,
	})
}

// @Summary Get currently present students
// @Description Lists all students currently present in the facility
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=[]StudentPresence} "Current students retrieved successfully"
// @Failure 500 {object} Response "Failed to fetch current students"
// @Router /dashboard/present [get]
func GetCurrentlyPresentStudents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	db, ok := MiddleWare.GetDB(r.Context())
	if !ok {
		http.Error(w, "DB not available", http.StatusInternalServerError)
		return
	}
	rows, err := db.Query(`
        SELECT s.rfid_id, s.name, s.department, a.in_time
        FROM students s
        JOIN attendance a ON s.id = a.student_id
        WHERE a.out_time IS NULL
        ORDER BY a.in_time
    `)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "Failed to fetch current students",
		})
		return
	}
	defer rows.Close()

	var students []Models.StudentPresence
	for rows.Next() {
		var rfid, name, department string
		var inTime time.Time
		err = rows.Scan(&rfid, &name, &department, &inTime)
		if err != nil {
			continue
		}
		students = append(students, Models.StudentPresence{
			RFID:       rfid,
			Name:       name,
			Department: department,
			InTime:     inTime.Format(time.RFC3339),
		})
	}

	json.NewEncoder(w).Encode(Models.Response{
		Status:  "success",
		Message: "Current students retrieved successfully",
		Data:    students,
	})
}

// @Summary Get attendance history
// @Description Retrieves complete attendance history for all students
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=[]AttendanceHistory} "Attendance history retrieved successfully"
// @Failure 500 {object} Response "Failed to fetch attendance history"
// @Router /dashboard/history [get]
func GetAttendanceHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	db, ok := MiddleWare.GetDB(r.Context())
	if !ok {
		http.Error(w, "DB not available", http.StatusInternalServerError)
		return
	}
	rows, err := db.Query(`
        SELECT s.rfid_id, s.name, a.in_time, a.out_time, a.total_duration
        FROM students s
        JOIN attendance a ON s.id = a.student_id
        ORDER BY a.in_time DESC
    `)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Models.Response{
			Status:  "error",
			Message: "Failed to fetch attendance history",
		})
		return
	}
	defer rows.Close()

	var history []Models.AttendanceHistory
	for rows.Next() {
		var rfid, name string
		var inTime, outTime sql.NullTime
		var totalDuration sql.NullString
		err = rows.Scan(&rfid, &name, &inTime, &outTime, &totalDuration)
		if err != nil {
			continue
		}
		history = append(history, Models.AttendanceHistory{
			RFID:     rfid,
			Name:     name,
			InTime:   inTime.Time.Format(time.RFC3339),
			OutTime:  outTime.Time.Format(time.RFC3339),
			Duration: totalDuration.String,
		})
	}

	json.NewEncoder(w).Encode(Models.Response{
		Status:  "success",
		Message: "Attendance history retrieved successfully",
		Data:    history,
	})
}
