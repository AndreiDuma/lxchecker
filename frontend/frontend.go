package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/AndreiDuma/lxchecker/scheduler"
)

// TODO: use gcfg for configuration?
// TODO: use flags instead of env variables?

var (
	mongo *mgo.Session
	sched *scheduler.Scheduler

	router = mux.NewRouter().StrictSlash(true)

	// TODO: make this key private.
	secure = securecookie.New([]byte("very-secret-hash"), nil)

	validSubjectId    = regexp.MustCompile(`[a-z]+[0-9a-z]+`)
	validAssignmentId = validSubjectId
)

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
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

func GetSubjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get subject id from request URL.
	subjectId := vars["subject_id"]
	if subjectId == "" {
		http.Error(w, "missing required `subject_id` field", http.StatusBadRequest)
		return
	}

	subject := Subject{}
	c := mongo.DB("lxchecker").C("subjects")
	if err := c.Find(nil).One(&subject); err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, "no subject matching given `subject_id`", http.StatusNotFound)
			return
		}
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, subject)

	assignments := []Assignment{}
	c = mongo.DB("lxchecker").C("assignments")
	if err := c.Find(bson.M{"subject_id": subjectId}).All(&assignments); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, a := range assignments {
		url, _ := router.Get("assignment").URL("subject_id", a.SubjectId, "assignment_id", a.Id)
		fmt.Fprintf(w, "Id: %v, Name: %v, Link: %v\n", a.Id, a.Name, url)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	subjects := []Subject{}
	c := mongo.DB("lxchecker").C("subjects")
	if err := c.Find(nil).All(&subjects); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, s := range subjects {
		fmt.Fprintf(w, "Id: %v, Name: %v\n", s.Id, s.Name)
	}
}

func CreateSubjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Get subject id from request URL.
	id := vars["subject_id"]
	if id == "" || !validSubjectId.MatchString(id) {
		http.Error(w, "bad or missing required `id` field", http.StatusBadRequest)
		return
	}

	// Get name from request params.
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "missing required `name` field", http.StatusBadRequest)
		return
	}

	// Insert subject in database.
	subject := &Subject{
		Id:   id,
		Name: name,
	}
	c := mongo.DB("lxchecker").C("subjects")
	if err := c.Insert(subject); err != nil {
		if mgo.IsDup(err) {
			http.Error(w, "subject with given `id` already exists", http.StatusBadRequest)
			return
		}
		log.Println("failed to insert subject:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
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

func LoggedInUser(r *http.Request) string {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return ""
	}

	var username string
	if err := secure.Decode("auth", cookie.Value, &username); err != nil {
		return ""
	}

	return username
}

func CurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, LoggedInUser(r))
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Get username from request.
	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "missing required `username` field", http.StatusBadRequest)
		return
	}

	// Get password from request.
	// TODO: use password hash instead of plain-text password.
	password := r.FormValue("password")
	if password == "" {
		http.Error(w, "missing required `password` field", http.StatusBadRequest)
		return
	}

	user := User{}
	c := mongo.DB("lxchecker").C("users")
	if err := c.Find(bson.M{"username": username, "password": password}).One(&user); err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Generate the auth token.
	token, err := secure.Encode("auth", username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	http.SetCookie(w, &http.Cookie{Name: "auth", Value: token})
	// TODO: redirect to `continue` parameter.
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
}

// User holds data related to an user.
type User struct {
	Username string
	Password string
}

// Subject holds data related to a subject.
type Subject struct {
	Id   string
	Name string
}

// Assignment holds data related to an assignment.
type Assignment struct {
	Id        string
	SubjectId string `bson:"subject_id"`

	Name           string
	Image          string
	Timeout        time.Duration
	SubmissionPath string `bson:"submission_path"`
}

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

func main() {
	var err error

	// Connect to Docker.
	if sched, err = scheduler.New(); err != nil {
		log.Fatalln(err)
	}

	// Connect to MongoDB.
	// TODO: customizable Mongo host
	mongo, err = mgo.Dial("localhost")
	if err != nil {
		log.Fatalln(err)
	}
	defer mongo.Close()

	// Ensure MongoDB indexes.
	if err = mongo.DB("lxchecker").C("assignments").EnsureIndex(mgo.Index{
		Key:    []string{"id", "subject_id"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `assignments`, key `shortname`")
	}
	if err = mongo.DB("lxchecker").C("subjects").EnsureIndex(mgo.Index{
		Key:    []string{"id"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `assignments`, key `shortname`")
	}
	if err = mongo.DB("lxchecker").C("users").EnsureIndex(mgo.Index{
		Key:    []string{"username"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `users`, key `username`")
	}

	// Setup handlers.
	router.HandleFunc("/login", LoginHandler)
	router.HandleFunc("/logout", LogoutHandler)
	router.HandleFunc("/current", CurrentUserHandler)

	s := router.PathPrefix("/-/").Subrouter()
	s.HandleFunc("/", IndexHandler).Methods("GET").Name("index")
	s.HandleFunc("/{subject_id}/", GetSubjectHandler).Methods("GET").Name("subject")
	s.HandleFunc("/{subject_id}/{assignment_id}/", GetAssignmentHandler).Methods("GET").Name("assignment")
	s.HandleFunc("/{subject_id}/{assignment_id}/{submission_id}/", GetSubmissionHandler).Methods("GET").Name("submission")

	s.HandleFunc("/{subject_id}/", CreateSubjectHandler).Methods("POST")
	s.HandleFunc("/{subject_id}/{assignment_id}/", CreateAssignmentHandler).Methods("POST")
	s.HandleFunc("/{subject_id}/{assignment_id}/{submission_id}/", SubmitHandler).Methods("POST")

	host := os.Getenv("LXCHECKER_FRONTEND_HOST")
	if host == "" {
		host = ":8080"
	}
	log.Printf("Listening on %s...\n", host)
	log.Fatalln(http.ListenAndServe(host, router))
}
