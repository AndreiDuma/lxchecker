package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"

	"github.com/AndreiDuma/lxchecker/scheduler"
)

// TODO: use gcfg for configuration?
// TODO: use flags instead of env variables?

var (
	mongo *mgo.Session
	sched *scheduler.Scheduler

	router = mux.NewRouter().StrictSlash(true)
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: add a proper index page instead of redirecting to login.
	http.Redirect(w, r, "/login", http.StatusFound)
}

func ListSubjectsHandler(w http.ResponseWriter, r *http.Request) {
	subjects := []Subject{}
	c := mongo.DB("lxchecker").C("subjects")
	if err := c.Find(nil).All(&subjects); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, s := range subjects {
		fmt.Fprintf(w, "Id: %v, Name: %v\n", s.Id, s.Name)
	}

	fmt.Fprintf(w, "\nUser: %v\n", GetUser(r).Username)
}

func main() {
	var err error

	// Connect to Docker.
	if sched, err = scheduler.New(); err != nil {
		log.Fatalln(err)
	}

	// Connect to MongoDB.
	// TODO: customizable Mongo host
	mongo, err = mgo.Dial("localhost")
	if err != nil {
		log.Fatalln(err)
	}
	defer mongo.Close()

	// Ensure MongoDB indexes.
	if err = mongo.DB("lxchecker").C("assignments").EnsureIndex(mgo.Index{
		Key:    []string{"id", "subject_id"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `assignments`, key `shortname`")
	}
	if err = mongo.DB("lxchecker").C("subjects").EnsureIndex(mgo.Index{
		Key:    []string{"id"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `assignments`, key `shortname`")
	}
	if err = mongo.DB("lxchecker").C("users").EnsureIndex(mgo.Index{
		Key:    []string{"username"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `users`, key `username`")
	}

	// Setup handlers.
	router.HandleFunc("/", IndexHandler).Methods("GET")
	router.HandleFunc("/login", LoginTmplHandler).Methods("GET")
	router.HandleFunc("/login", LoginHandler).Methods("POST")
	router.HandleFunc("/logout", LogoutHandler).Methods("GET") // TODO: make this POST.

	sub := router.PathPrefix("/-/").Subrouter()
	sub.Handle("/",
		RequireAuth(http.HandlerFunc(ListSubjectsHandler))).Methods("GET").Name("index")
	sub.Handle("/{subject_id}/",
		RequireAuth(http.HandlerFunc(GetSubjectHandler))).Methods("GET").Name("subject")
	sub.Handle("/{subject_id}/{assignment_id}/",
		RequireAuth(http.HandlerFunc(GetAssignmentHandler))).Methods("GET").Name("assignment")
	sub.Handle("/{subject_id}/{assignment_id}/{submission_id}/",
		RequireAuth(http.HandlerFunc(GetSubmissionHandler))).Methods("GET").Name("submission")

	sub.Handle("/{subject_id}/",
		RequireAuth(http.HandlerFunc(CreateSubjectHandler))).Methods("POST")
	sub.Handle("/{subject_id}/{assignment_id}/",
		RequireAuth(http.HandlerFunc(CreateAssignmentHandler))).Methods("POST")
	sub.Handle("/{subject_id}/{assignment_id}/{submission_id}/",
		RequireAuth(http.HandlerFunc(CreateSubmissionHandler))).Methods("POST")

	// TODO: receive this through command-line arguments.
	host := os.Getenv("LXCHECKER_FRONTEND_HOST")
	if host == "" {
		host = ":8080"
	}

	log.Printf("Listening on %s...\n", host)
	log.Fatalln(http.ListenAndServe(host, router))
}
