package main

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/gorilla/mux"
)

var (
	validSubjectId = regexp.MustCompile(`[a-z]+[0-9a-z]+`)
)

func CreateSubjectHandler(w http.ResponseWriter, r *http.Request) {
	s := db.Subject{}

	// Get id from request params.
	s.Id = r.FormValue("id")
	if s.Id == "" || !validSubjectId.MatchString(s.Id) {
		http.Error(w, "bad or missing required `id` field", http.StatusBadRequest)
		return
	}

	// Get name from request params.
	s.Name = r.FormValue("name")
	if s.Name == "" {
		http.Error(w, "missing required `name` field", http.StatusBadRequest)
		return
	}

	if err := db.InsertSubject(s); err != nil {
		if err == db.ErrAlreadyExists {
			http.Error(w, "subject with given `id` already exists", http.StatusBadRequest)
			return
		}
		panic(err)
	}
}

func GetSubjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get subject id from request URL.
	id := vars["id"]
	if id == "" {
		http.Error(w, "missing required `id` field", http.StatusBadRequest)
		return
	}

	subject, err := db.GetSubject(id)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no subject matching given `id`", http.StatusNotFound)
			return
		}
		panic(err)
	}

	fmt.Fprintln(w, subject)
	for _, a := range db.GetAllAssignments(id) {
		url, _ := router.Get("assignment").URL("subject_id", a.SubjectId, "id", a.Id)
		fmt.Fprintf(w, "Id: %v, Name: %v, Link: %v\n", a.Id, a.Name, url)
	}
}
