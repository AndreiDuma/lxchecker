package main

import (
	"html/template"
	"net/http"

	"github.com/AndreiDuma/lxchecker/db"
)

var (
	indexTmpl = template.Must(template.ParseFiles("templates/index.html"))
)

func LandingHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: add a proper index page instead of redirecting to login.
	http.Redirect(w, r, "/login", http.StatusFound)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	type D struct {
		Subject     *db.Subject
		Assignments []db.Assignment
	}
	data := []D{}
	for _, s := range db.GetAllSubjects() {
		data = append(data, D{&s, db.GetAllAssignments(s.Id)})
	}
	indexTmpl.Execute(w, data)
}
