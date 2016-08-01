package main

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/AndreiDuma/lxchecker/util"
	"github.com/gorilla/mux"
)

var (
	validAssignmentId = regexp.MustCompile(`[a-z]+[0-9a-z]+`)

	assignmentTmpl = template.Must(template.ParseFiles("templates/assignment.html"))
)

func CreateAssignmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get subject id from request URL.
	subjectId := vars["subject_id"]

	// Get assignment id from request URL.
	id := r.FormValue("id")

	// Get other attributes from request params.
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "missing required `name` field", http.StatusBadRequest)
		return
	}
	image := r.FormValue("image")
	if image == "" {
		http.Error(w, "missing required `image` field", http.StatusBadRequest)
		return
	}
	timeoutInt, err := strconv.Atoi(r.FormValue("timeout"))
	if err != nil {
		http.Error(w, "bad or missing required `timeout` field", http.StatusBadRequest)
		return
	}
	submission_path := r.FormValue("submission_path")
	if submission_path == "" {
		http.Error(w, "missing required `submission_path` field", http.StatusBadRequest)
		return
	}
	timeout := time.Duration(timeoutInt)

	// Insert assignment in database.
	a := db.Assignment{
		Id:             id,
		SubjectId:      subjectId,
		Name:           name,
		Image:          image,
		Timeout:        timeout,
		SubmissionPath: submission_path,
	}
	if err := db.InsertAssignment(a); err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no subject with given `subject_id`", http.StatusBadRequest)
			return
		}
		if err == db.ErrAlreadyExists {
			http.Error(w, "assignment with given `id` already exists", http.StatusBadRequest)
			return
		}
		panic(err)
	}

	// Redirect to the newly created assignment.
	http.Redirect(w, r, fmt.Sprintf("/-/%v/%v/", a.SubjectId, a.Id), http.StatusFound)
}

func GetAssignmentHandler(w http.ResponseWriter, r *http.Request) {
	rd := util.GetRequestData(r)

	subject, err := db.GetSubject(rd.SubjectId)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no subject matching given `subject_id`", http.StatusNotFound)
			return
		}
		panic(err)
	}
	assignment, err := db.GetAssignment(rd.SubjectId, rd.AssignmentId)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no assignment matching given `subject_id` and `id`", http.StatusNotFound)
			return
		}
		panic(err)
	}

	// Render template.
	type D struct {
		RequestData    *util.RequestData
		Subject        *db.Subject
		Assignment     *db.Assignment
		Submissions    []db.Submission
		AllSubmissions []db.Submission
	}
	assignmentTmpl.Execute(w, &D{
		rd,
		subject,
		assignment,
		db.GetAllSubmissionsOfUser(rd.SubjectId, rd.AssignmentId, rd.User.Username),
		db.GetAllSubmissions(rd.SubjectId, rd.AssignmentId),
	})
}
