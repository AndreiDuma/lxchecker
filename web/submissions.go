package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/AndreiDuma/lxchecker/db"
	"github.com/AndreiDuma/lxchecker/scheduler"
	"github.com/AndreiDuma/lxchecker/util"
	"golang.org/x/net/context"
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

	penalty := uint64(0)
	overdue := false
	now := time.Now()
	if now.After(assignment.SoftDeadline) && now.Before(assignment.HardDeadline) {
		daysLate := uint64(time.Now().Sub(assignment.SoftDeadline).Hours()/24) + 1
		penalty = daysLate * assignment.DailyPenalty
	} else if now.After(assignment.HardDeadline) {
		overdue = true
	}

	// Add submission to database.
	s := &db.Submission{
		Id:               db.NewSubmissionId(),
		AssignmentId:     rd.AssignmentId,
		SubjectId:        rd.SubjectId,
		OwnerUsername:    rd.User.Username,
		Timestamp:        time.Now(),
		UploadedFile:     submissionBytes,
		UploadedFileName: submissionFileHeader.Filename,
		Status:           "pending",
		Penalty:          penalty,
		Overdue:          overdue,
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
			return
		}

		// Store logs and metadata, then extract score.
		s.Logs = response.Logs
		s.Metadata = getMetadataFromLogs(response.Logs)
		if s.ScoreByTests, err = strconv.ParseUint(s.Metadata["SCORE"], 10, 64); err != nil {
			s.Status = "failed"
			db.UpdateSubmission(s)
			return
		}

		s.Status = "done"
		db.UpdateSubmission(s)
	}()

	// Redirect to the newly created submission.
	http.Redirect(w, r, fmt.Sprintf("/-/%v/%v/%v/", s.SubjectId, s.AssignmentId, s.Id), http.StatusFound)

}

func getMetadataFromLogs(logs []byte) map[string]string {
	metadata := map[string]string{}
	lines := bytes.Split(logs, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("@")) {
			continue
		}
		line = bytes.TrimLeft(line, "@")
		parts := bytes.SplitN(line, []byte(" "), 2)
		if len(parts) != 2 {
			continue
		}
		metadata[string(parts[0])] = string(parts[1])
	}
	return metadata
}

func getSubmissionHelper(w http.ResponseWriter, r *http.Request) *db.Submission {
	rd := util.GetRequestData(r)

	submission, err := db.GetSubmission(rd.SubjectId, rd.AssignmentId, rd.SubmissionId)
	if err != nil {
		if err == db.ErrNotFound {
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

		Subject    *db.Subject
		Assignment *db.Assignment
		Submission *db.Submission
	}
	submissionTmpl.Execute(w, &D{
		util.GetRequestData(r),
		db.GetSubjectOrPanic(s.SubjectId),
		db.GetAssignmentOrPanic(s.SubjectId, s.AssignmentId),
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

func GradeSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	s := getSubmissionHelper(w, r)
	if s == nil {
		return
	}

	// Get score from request params.
	var err error
	if s.ScoreByTeacher, err = strconv.ParseUint(r.FormValue("score"), 10, 64); err != nil {
		http.Error(w, "bad or missing required `score` field", http.StatusBadRequest)
		return
	}

	// Get feedback from request params.
	s.Feedback = r.FormValue("feedback")

	// Get grader username
	rd := util.GetRequestData(r)
	s.GraderUsername = rd.User.Username

	// Mark as graded.
	s.GradedByTeacher = true

	if err := db.UpdateSubmission(s); err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no submission matching given `subject_id`, `assignment_id` and `submission_id`", http.StatusNotFound)
			return
		}
		panic(err)
	}

	// Redirect back to the submission.
	http.Redirect(w, r, fmt.Sprintf("/-/%v/%v/%v/", s.SubjectId, s.AssignmentId, s.Id), http.StatusFound)
}
