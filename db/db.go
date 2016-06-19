package db

import (
	"errors"
	"log"

	"gopkg.in/mgo.v2"
)

var (
	mongo *mgo.Session

	ErrAlreadyExists = errors.New("object already exists")
	ErrNotFound      = errors.New("no such object")
)

func Init() {
	var err error
	if mongo, err = mgo.Dial("localhost"); err != nil {
		log.Fatalln("failed to connect to MongoDB")
	}

	// Ensure MongoDB indexes.
	if err = mongo.DB("lxchecker").C("subjects").EnsureIndex(mgo.Index{
		Key:    []string{"id"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `subjects`, key `id`")
	}
	if err = mongo.DB("lxchecker").C("assignments").EnsureIndex(mgo.Index{
		Key:    []string{"id", "subject_id"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `assignments`, keys `id` and `subject_id`")
	}
	if err = mongo.DB("lxchecker").C("submissions").EnsureIndex(mgo.Index{
		Key:    []string{"id", "assignment_id", "subject_id"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `submissions`, keys `id`, `assignment_id` and `subject_id`")
	}
	if err = mongo.DB("lxchecker").C("users").EnsureIndex(mgo.Index{
		Key:    []string{"username"},
		Unique: true,
	}); err != nil {
		log.Fatalln("failed to ensure an unique index on collection `users`, key `username`")
	}
}

func Done() {
	defer mongo.Close()
}
