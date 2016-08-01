package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/AndreiDuma/lxchecker/scheduler"
	"github.com/AndreiDuma/lxchecker/util"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
)

var (
	submissionTmpl = template.Must(template.ParseFiles("templates/submission.html"))
)

func CreateSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	rd := util.GetRequestData(r)

	// Get submission file from request.
	submissionFile, submissionFileHeader, err := r.FormFile("submission")
	if err != nil {
		http.Error(w, "missing required `submission` field", http.StatusBadRequest)
		return
	}
	submissionBytes, err := ioutil.ReadAll(submissionFile)
	if err != nil {
		panic(err)
	}

	assignment, err := db.GetAssignment(rd.SubjectId, rd.AssignmentId)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no assignment matching given `subject_id` and `assignment_id`", http.StatusBadRequest)
			return
		}
		panic(err)
	}

	// Add submission to database.
	s := &db.Submission{
		Id:               db.NewSubmissionId(),
		AssignmentId:     rd.AssignmentId,
		SubjectId:        rd.SubjectId,
		OwnerUsername:    util.CurrentUser(r).Username,
		Timestamp:        time.Now(),
		UploadedFile:     submissionBytes,
		UploadedFileName: submissionFileHeader.Filename,
		Status:           "pending",
	}
	if err = db.InsertSubmission(s); err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no assignment matching given `subject_id` and `assignment_id`", http.StatusBadRequest)
			return
		}
		panic(err)
	}

	// Do the actual testing in a separate goroutine.
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

		// Store logs and update status.
		s.Logs = response.Logs
		s.Status = "done"

		db.UpdateSubmission(s)
	}()

	// Redirect to the newly created submission.
	http.Redirect(w, r, fmt.Sprintf("/-/%v/%v/%v/", s.SubjectId, s.AssignmentId, s.Id), http.StatusFound)

}

func getSubmissionHelper(w http.ResponseWriter, r *http.Request) *db.Submission {
	rd := util.GetRequestData(r)

	// Get submission hex id from request URL.
	if rd.SubmissionId == "" {
		http.Error(w, "missing required `submission_id` field", http.StatusBadRequest)
		return nil
	}

	submission, err := db.GetSubmission(rd.SubjectId, rd.AssignmentId, rd.SubmissionId)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, "no submission matching given `subject_id`, `assignment_id` and `submission_id`", http.StatusNotFound)
			return nil
		}
		panic(err)
	}

	return submission
}

func GetSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	s := getSubmissionHelper(w, r)
	if s == nil {
		return
	}

	// Render template.
	type D struct {
		RequestData *util.RequestData

		/*
			Subject     *db.Subject
			Assignment  *db.Assignment
		*/
		Submission *db.Submission
	}
	submissionTmpl.Execute(w, &D{
		util.GetRequestData(r),
		/*
			subject,
			assignment,
		*/
		s,
	})
}

func GetSubmissionUploadHandler(w http.ResponseWriter, r *http.Request) {
	s := getSubmissionHelper(w, r)
	if s == nil {
		return
	}
	// TODO: make the downloaded submission have at least the same extension as the uploaded one.
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%v"`, s.UploadedFileName))
	w.Write(s.UploadedFile)
}
