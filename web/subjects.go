package main

import (
	"html/template"
	"net/http"
	"regexp"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/gorilla/mux"
)

var (
	validSubjectId = regexp.MustCompile(`[a-z]+[0-9a-z]+`)

	subjectTmpl = template.Must(template.ParseFiles("templates/subject.html"))
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

	// Render template.
	type D struct {
		Subject     *db.Subject
		Assignments []db.Assignment
	}
	subjectTmpl.Execute(w, &D{subject, db.GetAllAssignments(subject.Id)})
}
