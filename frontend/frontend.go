package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/AndreiDuma/lxchecker/scheduler"
)

// TODO: use gcfg for configuration?
// TODO: use flags instead of env variables?

var (
	session *mgo.Session
)

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	// get submission file from request
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

	// TODO: get image name from db using field `assignment_id`
	image := "andreiduma/lxchecker_so_tema3"

	// TODO: get extraction path from db
	submissionPath := "/submission/submission.zip"

	// TODO: get submission timeout from db
	timeout := 60 * time.Second

	// add submission to database
	submission := &Submission{
		Id:           bson.NewObjectId(),
		UploadedFile: submissionBytes,
		Status:       "pending",
	}
	c := session.DB("lxchecker").C("submissions")
	if err = c.Insert(submission); err != nil {
		log.Println("failed to insert submission:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	go func() {
		// submit for testing
		options := scheduler.SubmitOptions{
			Image:          image,
			Submission:     submissionBytes,
			SubmissionPath: submissionPath,
			Timeout:        timeout,
		}
		response, err := scheduler.Submit(context.Background(), options)
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

		// store the logs
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

		// success
		c := session.DB("lxchecker").C("submissions")
		if err := c.UpdateId(submission.Id, bson.M{
			"$set": bson.M{"status": "done"},
		}); err != nil {
			log.Println("submission status couldn't be updated to `done`:", err)
		}
	}()
}

func ResultHandler(w http.ResponseWriter, r *http.Request) {
	idHex := r.FormValue("id")
	if idHex == "" || !bson.IsObjectIdHex(idHex) {
		http.Error(w, "bad or missing required `id` field", http.StatusBadRequest)
		return
	}
	fmt.Println(idHex)
	id := bson.ObjectIdHex(idHex)
	fmt.Println(id)

	submission := Submission{}
	c := session.DB("lxchecker").C("submissions")
	if err := c.FindId(id).One(&submission); err != nil {
		// submission not found
		if err == mgo.ErrNotFound {
			http.Error(w, "no submission matching given id", http.StatusNotFound)
			return
		}
		// another error
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(submission.Logs); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func ListSubmissionsHandler(w http.ResponseWriter, req *http.Request) {
	submissions := []Submission{}
	c := session.DB("lxchecker").C("submissions")
	if err := c.Find(nil).All(&submissions); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, s := range submissions {
		fmt.Fprintf(w, "Id: %v, Status: %v\n", s.Id, s.Status)
	}
}

// Submission holds data related to a submission.
type Submission struct {
	Id           bson.ObjectId `bson:"_id"`
	UploadedFile []byte
	Status       string // TODO: make this a constant or an enum.
	Score        uint
	Feedback     string
	Logs         []byte
}

func main() {
	var err error
	if err = scheduler.Init(); err != nil {
		log.Fatalln(err)
	}

	// setup MongoDB
	session, err = mgo.Dial("localhost")
	if err != nil {
		log.Fatalln(err)
	}
	defer session.Close()

	// setup handlers
	http.HandleFunc("/submit", SubmitHandler)
	http.HandleFunc("/result", ResultHandler)
	http.HandleFunc("/", ListSubmissionsHandler)

	host := os.Getenv("LXCHECKER_FRONTEND_HOST")
	if host == "" {
		host = ":8080"
	}
	log.Printf("Listening on %s...\n", host)
	log.Fatalln(http.ListenAndServe(host, nil))
}
