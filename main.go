package main

import (
	"context"
	"database/sql"
	_ "encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "time"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	
	// Replace it with your actual module path
	"rfidattendance/Handler"
	"rfidattendance/MiddleWare"
	_ "rfidattendance/docs"
)

var db *sql.DB

// @title IoT Attendance System API
// @version 1.0
// @description This is an attendance tracking system API using RFID
// @host localhost:8080
// @BasePath /
func main() {
	var err error

	db, err = sql.Open("pgx", "postgresql://neondb_owner:npg_aSm2XKLDbu5J@ep-nameless-grass-a1hhakn0-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal("Failed to close database connection:", err)
		}
	}(db)

	ctxx := context.Background()
	ctx := MiddleWare.AttachDB(ctxx, db)

	r := mux.NewRouter()

	// Swagger documentation endpoint
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// Register routes
	r.HandleFunc("/walkin/{rfid}", MiddleWare.DBMiddleware(ctx, Handler.WalkIn)).Methods("POST")
	r.HandleFunc("/walkout/{rfid}", MiddleWare.DBMiddleware(ctx, Handler.WalkOut)).Methods("POST")
	r.HandleFunc("/student/{rfid}/attendance", MiddleWare.DBMiddleware(ctx, Handler.GetStudentAttendance)).Methods("GET")
	r.HandleFunc("/dashboard/present", MiddleWare.DBMiddleware(ctx, Handler.GetCurrentlyPresentStudents)).Methods("GET")
	r.HandleFunc("/dashboard/history", MiddleWare.DBMiddleware(ctx, Handler.GetAttendanceHistory)).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	// Start server
	handler := c.Handler(r)
	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
