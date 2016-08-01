package db

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Teacher describes the teacher of a User on a Subject.
type Teacher struct {
	Username  string
	SubjectId string `bson:"subject_id"`
}

func GetAllTeachersOfSubject(subjectId string) []User {
	teachers := []User{}
	c := mongo.DB("lxchecker").C("teachers")
	if err := c.Find(bson.M{"subject_id": subjectId}).All(&teachers); err != nil {
		panic(err)
	}
	return teachers
}

func IsTeacher(username, subjectId string) bool {
	user := User{}
	c := mongo.DB("lxchecker").C("teachers")
	if err := c.Find(bson.M{"username": username, "subject_id": subjectId}).One(&user); err != nil {
		if err == mgo.ErrNotFound {
			return false
		}
		panic(err)
	}
	return true
}

func InsertTeacher(t *Teacher) error {
	c := mongo.DB("lxchecker").C("teachers")
	if err := c.Insert(t); err != nil {
		if mgo.IsDup(err) {
			return ErrAlreadyExists
		}
		panic(err)
	}
	return nil
}
