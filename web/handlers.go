package main

import (
	"html/template"
	"net/http"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/AndreiDuma/lxchecker/util"
)

var (
	indexTmpl = template.Must(template.ParseFiles("templates/index.html"))
)

func LandingHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: add a proper index page instead of redirecting to login.
	http.Redirect(w, r, "/login", http.StatusFound)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	type S struct {
		Subject     db.Subject
		Assignments []db.Assignment
	}
	type D struct {
		RequestData *util.RequestData

		Subjects []S
		Admins   []db.User
	}

	data := D{
		RequestData: util.GetRequestData(r),
		Subjects:    []S{},
		Admins:      db.GetAdmins(),
	}
	for _, s := range db.GetAllSubjects() {
		data.Subjects = append(data.Subjects, S{s, db.GetAllAssignments(s.Id)})
	}
	indexTmpl.Execute(w, data)
}

func AddAdminHandler(w http.ResponseWriter, r *http.Request) {
	// Get user to make admin from request params.
	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "missing required `username` field", http.StatusBadRequest)
		return
	}
	u, err := db.GetUser(username)
	if err == db.ErrNotFound {
		http.Error(w, "no user matching given `username`", http.StatusNotFound)
		return
	}

	u.IsAdmin = true
	if db.UpdateUser(u) == db.ErrNotFound {
		http.Error(w, "no user matching given `username`", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/-/", http.StatusFound)
}
