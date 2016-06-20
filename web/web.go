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

// TODO: use gcfg for configuration?
// TODO: use flags instead of env variables?

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
	// TODO: wrap router with gorrila/handlers/recovery handler.
	router.HandleFunc("/", LandingHandler).Methods("GET")
	router.HandleFunc("/login", LoginTmplHandler).Methods("GET")
	router.HandleFunc("/login", LoginHandler).Methods("POST")
	router.HandleFunc("/signup", SignupTmplHandler).Methods("GET")
	router.HandleFunc("/signup", SignupHandler).Methods("POST")
	router.HandleFunc("/logout", LogoutHandler).Methods("GET") // TODO: make this POST.

	sub := router.PathPrefix("/-/").Subrouter()
	sub.Handle("/", util.RequireAuth(http.HandlerFunc(IndexHandler))).Methods("GET").Name("index")
	sub.Handle("/{id}/", util.RequireAuth(http.HandlerFunc(GetSubjectHandler))).Methods("GET").Name("subject")
	sub.Handle("/{subject_id}/{id}/", util.RequireAuth(http.HandlerFunc(GetAssignmentHandler))).Methods("GET").Name("assignment")
	sub.Handle("/{subject_id}/{assignment_id}/{id}/", util.RequireAuth(http.HandlerFunc(GetSubmissionHandler))).Methods("GET").Name("submission")

	sub.Handle("/create_subject/", util.RequireAuth(http.HandlerFunc(CreateSubjectHandler))).Methods("POST")
	sub.Handle("/{subject_id}/create_assignment/", util.RequireAuth(http.HandlerFunc(CreateAssignmentHandler))).Methods("POST")
	sub.Handle("/{subject_id}/{assignment_id}/create_submission/", util.RequireAuth(http.HandlerFunc(CreateSubmissionHandler))).Methods("POST")

	// TODO: receive this through command-line arguments.
	host := os.Getenv("LXCHECKER_FRONTEND_HOST")
	if host == "" {
		host = ":8080"
	}

	log.Printf("Listening on %s...\n", host)
	log.Fatalln(http.ListenAndServe(host, router))
}
