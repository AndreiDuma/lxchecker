package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/gorilla/mux"
)

var (
	validAssignmentId = regexp.MustCompile(`[a-z]+[0-9a-z]+`)

	ErrAssignmentExists = errors.New("assignment already exists")
	ErrNoSuchAssignment = errors.New("no such assignment")
)

func CreateAssignmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get subject id from request URL.
	subjectId := vars["subject_id"]
	if subjectId == "" {
		http.Error(w, "missing required `subject_id` field", http.StatusBadRequest)
		return
	}

	// Get assignment id from request URL.
	id := r.FormValue("id")
	if id == "" || !validAssignmentId.MatchString(id) {
		http.Error(w, "bad or missing required `id` field", http.StatusBadRequest)
		return
	}

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
}

func GetAssignmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get subject id from request URL.
	subjectId := vars["subject_id"]
	if subjectId == "" {
		http.Error(w, "missing required `subject_id` field", http.StatusBadRequest)
		return
	}

	// Get assignment id from request URL.
	id := vars["id"]
	if id == "" {
		http.Error(w, "missing required `id` field", http.StatusBadRequest)
		return
	}

	submissions := db.GetAllSubmissions(subjectId, id)
	b, err := json.Marshal(submissions)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}
