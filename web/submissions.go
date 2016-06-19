package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/AndreiDuma/lxchecker/scheduler"
	"github.com/AndreiDuma/lxchecker/util"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
)

func CreateSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get subject & assignment ids from request URL.
	subjectId := vars["subject_id"]
	if subjectId == "" {
		http.Error(w, "missing required `subject_id` field", http.StatusBadRequest)
		return
	}
	assignmentId := vars["assignment_id"]
	if assignmentId == "" {
		http.Error(w, "missing required `assignment_id` field", http.StatusBadRequest)
		return
	}

	// Get submission file from request.
	submissionFile, _, err := r.FormFile("submission")
	if err != nil {
		http.Error(w, "missing required `submission` field", http.StatusBadRequest)
		return
	}
	submissionBytes, err := ioutil.ReadAll(submissionFile)
	if err != nil {
		panic(err)
	}

	assignment, err := db.GetAssignment(subjectId, assignmentId)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no assignment matching given `subject_id` and `assignment_id`", http.StatusBadRequest)
			return
		}
		panic(err)
	}

	// Add submission to database.
	s := db.Submission{
		Id:           db.NewSubmissionId(),
		AssignmentId: assignmentId,
		SubjectId:    subjectId,
		Timestamp:    time.Now(),
		UploadedFile: submissionBytes,
		Status:       "pending",
	}
	if err = db.InsertSubmission(s); err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no assignment matching given `subject_id` and `assignment_id`", http.StatusBadRequest)
			return
		}
		if err == db.ErrAlreadyExists {
			http.Error(w, "submission with given `id` already exists", http.StatusBadRequest)
			return
		}
		panic(err)
	}

	go func() {
		defer util.LogPanics()

		// Submit for testing.
		options := scheduler.SubmitOptions{
			Image:          assignment.Image,
			Submission:     submissionBytes,
			SubmissionPath: assignment.SubmissionPath,
			Timeout:        assignment.Timeout * time.Second,
		}
		response, err := sched.Submit(context.Background(), options)
		if err != nil {
			s.Status = "failed"
			db.UpdateSubmission(s)
		}

		// Store the logs.
		if s.Logs, err = ioutil.ReadAll(response.Logs); err != nil {
			s.Status = "failed"
			db.UpdateSubmission(s)
		}
		s.Status = "done"
		db.UpdateSubmission(s)
	}()
}

func GetSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get subject id from request URL.
	subjectId := vars["subject_id"]
	if subjectId == "" {
		http.Error(w, "missing required `subject_id` field", http.StatusBadRequest)
		return
	}

	// Get assignment id from request URL.
	assignmentId := vars["assignment_id"]
	if assignmentId == "" {
		http.Error(w, "missing required `assignment_id` field", http.StatusBadRequest)
		return
	}

	// Get submission hex id from request URL.
	id := vars["id"]
	if id == "" {
		http.Error(w, "missing required `id` field", http.StatusBadRequest)
		return
	}

	submission, err := db.GetSubmission(subjectId, assignmentId, id)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, "no submission matching given `subject_id`, `assignment_id` and `id`", http.StatusNotFound)
			return
		}
		panic(err)
	}

	if _, err := fmt.Fprintln(w, submission); err != nil {
		panic(err)
	}
}
