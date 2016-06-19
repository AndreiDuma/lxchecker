package web

import (
	"fmt"
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
	// TODO: feed a template with the subjects.
	db.GetAllSubjects()
}
