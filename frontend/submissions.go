package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/AndreiDuma/lxchecker/scheduler"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Submission holds data related to a submission.
type Submission struct {
	Id           bson.ObjectId `bson:"_id"`
	AssignmentId string        `bson:"assignment_id"`
	SubjectId    string        `bson:"subject_id"`

	Status       string // TODO: make this a constant or an enum.
	Timestamp    time.Time
	UploadedFile []byte `bson:"uploaded_file",json:"-"`
	Logs         []byte
	Score        uint
	Feedback     string
}

func CreateSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	// Get submission file from request.
	submissionFile, _, err := r.FormFile("submission")
	if err != nil {
		http.Error(w, "missing required `submission` field", http.StatusBadRequest)
		return
	}
	submissionBytes, err := ioutil.ReadAll(submissionFile)
	if err != nil {
		log.Println("failed to read uploaded file:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Get subject & assignment ids from request.
	subjectId := r.FormValue("subject_id")
	if subjectId == "" {
		http.Error(w, "missing required `subject_id` field", http.StatusBadRequest)
		return
	}
	assignmentId := r.FormValue("assignment_id")
	if assignmentId == "" {
		http.Error(w, "missing required `assignment_id` field", http.StatusBadRequest)
		return
	}

	assignment := Assignment{}
	c := mongo.DB("lxchecker").C("assignments")
	if err := c.Find(bson.M{"id": assignmentId, "subject_id": subjectId}).One(&assignment); err != nil {
		// Submission not found.
		if err == mgo.ErrNotFound {
			http.Error(w, "no assignment matching given assignment_id", http.StatusNotFound)
			return
		}
		// Another error.
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Add submission to database.
	submission := &Submission{
		Id:           bson.NewObjectId(),
		AssignmentId: assignmentId,
		SubjectId:    subjectId,
		Timestamp:    time.Now(),
		UploadedFile: submissionBytes,
		Status:       "pending",
	}
	c = mongo.DB("lxchecker").C("submissions")
	if err = c.Insert(submission); err != nil {
		log.Println("failed to insert submission:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	go func() {
		// Submit for testing.
		options := scheduler.SubmitOptions{
			Image:          assignment.Image,
			Submission:     submissionBytes,
			SubmissionPath: assignment.SubmissionPath,
			Timeout:        assignment.Timeout * time.Second,
		}
		response, err := sched.Submit(context.Background(), options)
		if err != nil {
			log.Println("testing failed:", err)
			// TODO: factor out status update code.
			if err := c.UpdateId(submission.Id, bson.M{
				"$set": bson.M{"status": "failed"},
			}); err != nil {
				log.Println("submission status couldn't be updated to `failed`:", err)
			}
			return
		}

		// Store the logs.
		logs, err := ioutil.ReadAll(response.Logs)
		if err != nil {
			log.Println("reading logs failed:", err)
			if err := c.UpdateId(submission.Id, bson.M{
				"$set": bson.M{"status": "failed"},
			}); err != nil {
				log.Println("submission status couldn't be updated to `failed`:", err)
			}
			return
		}
		if err := c.UpdateId(submission.Id, bson.M{
			"$set": bson.M{"logs": logs},
		}); err != nil {
			log.Println("storing logs in db failed:", err)
			if err := c.UpdateId(submission.Id, bson.M{
				"$set": bson.M{"status": "failed"},
			}); err != nil {
				log.Println("submission status couldn't be updated to `failed`:", err)
			}
			return
		}

		// Success.
		c := mongo.DB("lxchecker").C("submissions")
		if err := c.UpdateId(submission.Id, bson.M{
			"$set": bson.M{"status": "done"},
		}); err != nil {
			log.Println("submission status couldn't be updated to `done`:", err)
		}
	}()
}

func GetSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get submission hex id from request URL.
	idHex := vars["submission_id"]
	if idHex == "" || !bson.IsObjectIdHex(idHex) {
		http.Error(w, "bad or missing required `id` field", http.StatusBadRequest)
		return
	}
	id := bson.ObjectIdHex(idHex)

	submission := Submission{}
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.FindId(id).One(&submission); err != nil {
		// Submission not found.
		if err == mgo.ErrNotFound {
			http.Error(w, "no submission matching given `id`", http.StatusNotFound)
			return
		}
		// Another error.
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprintln(w, submission); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
