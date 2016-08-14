package db

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Subject holds data related to a subject.
type Subject struct {
	Id   string
	Name string
}

func GetSubject(id string) (*Subject, error) {
	s := Subject{}
	c := mongo.DB("lxchecker").C("subjects")
	if err := c.Find(bson.M{"id": id}).One(&s); err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		panic(err)
	}
	return &s, nil
}

func GetSubjectOrPanic(id string) *Subject {
	subject, err := GetSubject(id)
	if err != nil {
		panic(err)
	}
	return subject
}

func GetAllSubjects() []Subject {
	subjects := []Subject{}
	c := mongo.DB("lxchecker").C("subjects")
	if err := c.Find(nil).All(&subjects); err != nil {
		panic(err)
	}
	return subjects
}

func InsertSubject(s Subject) error {
	c := mongo.DB("lxchecker").C("subjects")
	if err := c.Insert(s); err != nil {
		if mgo.IsDup(err) {
			return ErrAlreadyExists
		}
		panic(err)
	}
	return nil
}
