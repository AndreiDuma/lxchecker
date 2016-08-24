package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/AndreiDuma/lxchecker/scheduler"
	"github.com/AndreiDuma/lxchecker/util"
)

// TODO: use flags instead of environment variables.

var (
	sched *scheduler.Scheduler

	router = mux.NewRouter().StrictSlash(true)
)

func main() {
	// Connect to Docker.
	sched = scheduler.New()

	// Connect to MongoDB.
	// TODO: customizable Mongo host.
	db.Init()
	defer db.Done()

	// Setup handlers.
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	// TODO: wrap router with gorrila/handlers/recovery handler.
	router.HandleFunc("/", LandingHandler).Methods("GET")
	router.HandleFunc("/login", LoginTmplHandler).Methods("GET")
	router.HandleFunc("/signup", SignupTmplHandler).Methods("GET")

	router.HandleFunc("/login", LoginHandler).Methods("POST")
	router.HandleFunc("/signup", SignupHandler).Methods("POST")
	router.HandleFunc("/logout", LogoutHandler).Methods("GET") // TODO: make this POST.
	router.Handle("/add_admin", util.RequireAuth(util.RequireAdmin(http.HandlerFunc(AddAdminHandler)))).Methods("POST")

	sub := router.PathPrefix("/-/").Subrouter()
	sub.Handle("/", util.RequireAuth(http.HandlerFunc(IndexHandler))).Methods("GET")
	sub.Handle("/{subject_id}/", util.RequireAuth(http.HandlerFunc(GetSubjectHandler))).Methods("GET")
	sub.Handle("/{subject_id}/{assignment_id}/", util.RequireAuth(http.HandlerFunc(GetAssignmentHandler))).Methods("GET")
	sub.Handle("/{subject_id}/{assignment_id}/{submission_id}/", util.RequireAuth(http.HandlerFunc(GetSubmissionHandler))).Methods("GET")
	sub.Handle("/{subject_id}/{assignment_id}/{submission_id}/upload", util.RequireAuth(http.HandlerFunc(GetSubmissionUploadHandler))).Methods("GET")

	sub.Handle("/create_subject", util.RequireAuth(util.RequireAdmin(http.HandlerFunc(CreateSubjectHandler)))).Methods("POST")
	sub.Handle("/{subject_id}/create_assignment", util.RequireAuth(util.RequireTeacherOrAdmin(http.HandlerFunc(CreateAssignmentHandler)))).Methods("POST")
	sub.Handle("/{subject_id}/add_teacher", util.RequireAuth(util.RequireTeacherOrAdmin(http.HandlerFunc(AddTeacherHandler)))).Methods("POST")
	sub.Handle("/{subject_id}/{assignment_id}/create_submission", util.RequireAuth(http.HandlerFunc(CreateSubmissionHandler))).Methods("POST")
	sub.Handle("/{subject_id}/{assignment_id}/{submission_id}/grade_submission", util.RequireAuth(util.RequireTeacherOrAdmin(http.HandlerFunc(GradeSubmissionHandler)))).Methods("POST")

	// TODO: receive this through a command-line argument.
	host := os.Getenv("LXCHECKER_FRONTEND_HOST")
	if host == "" {
		host = ":8080"
	}

	log.Printf("Listening on %s...\n", host)
	log.Fatalln(http.ListenAndServe(host, router))
}
