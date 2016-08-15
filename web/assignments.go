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
)

var (
	validAssignmentId  = regexp.MustCompile(`[a-z]+[0-9a-z]+`)
	deadlineDateFormat = "02.01.2006"

	assignmentTmpl = template.Must(template.ParseFiles("templates/assignment.html"))
)

func CreateAssignmentHandler(w http.ResponseWriter, r *http.Request) {
	rd := util.GetRequestData(r)

	// Get assignment id from request params.
	assignmentId := r.FormValue("assignment_id")

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
	timeout := time.Duration(timeoutInt)

	submissionPath := r.FormValue("submission_path")
	if submissionPath == "" {
		http.Error(w, "missing required `submission_path` field", http.StatusBadRequest)
		return
	}

	// The deadlines are actually at the end of the day.
	getEndOfDay := func(t time.Time) time.Time {
		return t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	softDeadline, err := time.Parse(deadlineDateFormat, r.FormValue("soft_deadline"))
	if err != nil {
		http.Error(w, "bad or missing required `soft_deadline` field", http.StatusBadRequest)
		return
	}
	softDeadline = getEndOfDay(softDeadline)
	hardDeadline, err := time.Parse(deadlineDateFormat, r.FormValue("hard_deadline"))
	if err != nil {
		http.Error(w, "bad or missing required `hard_deadline` field", http.StatusBadRequest)
		return
	}
	hardDeadline = getEndOfDay(hardDeadline)
	dailyPenalty, err := strconv.Atoi(r.FormValue("daily_penalty"))
	if err != nil {
		http.Error(w, "bad or missing required `daily_penalty` field", http.StatusBadRequest)
		return
	}

	// Insert assignment in database.
	a := db.Assignment{
		Id:             assignmentId,
		SubjectId:      rd.SubjectId,
		Name:           name,
		Image:          image,
		Timeout:        timeout,
		SubmissionPath: submissionPath,
		SoftDeadline:   softDeadline,
		HardDeadline:   hardDeadline,
		DailyPenalty:   dailyPenalty,
	}
	if err := db.InsertAssignment(a); err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "no subject with given `subject_id`", http.StatusBadRequest)
			return
		}
		if err == db.ErrAlreadyExists {
			http.Error(w, "assignment with given `assignment_id` already exists", http.StatusBadRequest)
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
		RequestData       *util.RequestData
		Subject           *db.Subject
		Assignment        *db.Assignment
		Submissions       []db.Submission
		ActiveSubmissions []db.Submission
		AllSubmissions    []db.Submission
	}
	assignmentTmpl.Execute(w, &D{
		rd,
		subject,
		assignment,
		db.GetSubmissionsOfUser(rd.SubjectId, rd.AssignmentId, rd.User.Username),
		db.GetActiveSubmissions(rd.SubjectId, rd.AssignmentId),
		db.GetAllSubmissions(rd.SubjectId, rd.AssignmentId),
	})
}
