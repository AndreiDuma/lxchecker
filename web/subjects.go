package web

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/AndreiDuma/lxchecker/util"
)

var (
	validSubjectId = regexp.MustCompile(`[a-z]+[0-9a-z]+`)

	subjectTmpl = template.Must(template.ParseFiles("templates/base.html", "templates/subject.html"))
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

	// Insert subject in database.
	if err := db.InsertSubject(s); err != nil {
		if err == db.ErrAlreadyExists {
			http.Error(w, "subject with given `id` already exists", http.StatusBadRequest)
			return
		}
		panic(err)
	}

	// Redirect to the newly created subject.
	http.Redirect(w, r, fmt.Sprintf("/-/%v/", s.Id), http.StatusFound)
}

func GetSubjectHandler(w http.ResponseWriter, r *http.Request) {
	rd := util.GetRequestData(r)

	// Get subject id from request URL.
	if rd.SubjectId == "" {
		http.Error(w, "missing required `subject_id` field", http.StatusBadRequest)
		return
	}

	subject, err := db.GetSubject(rd.SubjectId)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no subject matching given `subject_id`", http.StatusNotFound)
			return
		}
		panic(err)
	}

	// Render template.
	type D struct {
		RequestData *util.RequestData

		Subject     *db.Subject
		Assignments []db.Assignment
		Teachers    []db.User
	}
	subjectTmpl.Execute(w, &D{
		rd,
		subject,
		db.GetAllAssignments(subject.Id),
		db.GetAllTeachersOfSubject(subject.Id),
	})
}

func AddTeacherHandler(w http.ResponseWriter, r *http.Request) {
	rd := util.GetRequestData(r)
	t := &db.Teacher{}

	// Get subject id from request.
	t.SubjectId = rd.SubjectId

	// Get username from request params.
	t.Username = r.FormValue("username")
	if t.Username == "" {
		http.Error(w, "missing required `username` field", http.StatusBadRequest)
		return
	}

	if _, err := db.GetUser(t.Username); err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no user with given `username`", http.StatusNotFound)
			return
		}
	}

	// Insert teacher role in database.
	if err := db.InsertTeacher(t); err != nil {
		if err == db.ErrAlreadyExists {
			http.Error(w, "user with given `username` is already a teacher", http.StatusBadRequest)
			return
		}
		panic(err)
	}
	http.Redirect(w, r, fmt.Sprintf("/-/%v/", t.SubjectId), http.StatusFound)
}
