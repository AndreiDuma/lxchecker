package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	validAssignmentId = regexp.MustCompile(`[a-z]+[0-9a-z]+`)
)

// Assignment holds data related to an assignment.
type Assignment struct {
	Id        string
	SubjectId string `bson:"subject_id"`

	Name           string
	Image          string
	Timeout        time.Duration
	SubmissionPath string `bson:"submission_path"`
}

func CreateAssignmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get assignment id from request URL.
	id := vars["assignment_id"]
	if id == "" || !validAssignmentId.MatchString(id) {
		http.Error(w, "bad or missing required `id` field", http.StatusBadRequest)
		return
	}

	// Get subject id from request URL and check it exists.
	subjectId := vars["subject_id"]
	if subjectId == "" {
		http.Error(w, "missing required `subject_id` field", http.StatusBadRequest)
		return
	}
	c := mongo.DB("lxchecker").C("subjects")
	if n, err := c.Find(bson.M{"id": subjectId}).Count(); err != nil || n == 0 {
		http.Error(w, "no subject with given `subject_id`", http.StatusNotFound)
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
	assignment := &Assignment{
		Id:             id,
		SubjectId:      subjectId,
		Name:           name,
		Image:          image,
		Timeout:        timeout,
		SubmissionPath: submission_path,
	}
	c = mongo.DB("lxchecker").C("assignments")
	if err := c.Insert(assignment); err != nil {
		if mgo.IsDup(err) {
			http.Error(w, "assignment with given `id` already exists", http.StatusBadRequest)
			return
		}
		log.Println("failed to insert subject:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
	assignmentId := vars["assignment_id"]
	if assignmentId == "" {
		http.Error(w, "missing required `assignment_id` field", http.StatusBadRequest)
		return
	}

	submissions := []Submission{}
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Find(bson.M{"subject_id": subjectId, "assignment_id": assignmentId}).All(&submissions); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	/*
		for _, s := range submissions {
			fmt.Fprintf(w, "Id: %v, Timestamp: %v, Status: %v\n", s.Id, s.Timestamp, s.Status)
		}
	*/
	b, err := json.Marshal(submissions)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}
