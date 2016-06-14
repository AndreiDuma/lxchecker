package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Subject holds data related to a subject.
type Subject struct {
	Id   string
	Name string
}

var (
	validSubjectId = regexp.MustCompile(`[a-z]+[0-9a-z]+`)
)

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
