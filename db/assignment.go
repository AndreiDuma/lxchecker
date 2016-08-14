package db

import (
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Assignment holds data related to an assignment.
type Assignment struct {
	Id        string
	SubjectId string `bson:"subject_id"`

	Name           string
	Image          string
	Timeout        time.Duration
	SubmissionPath string `bson:"submission_path"`

	SoftDeadline time.Time `bson:"soft_deadline"`
	HardDeadline time.Time `bson:"hard_deadline"`
	DailyPenalty uint64    `bson:"daily_penalty"`

	MaxScoreByTests   uint64 `bson:"max_score_by_tests"`
	MaxScoreByTeacher uint64 `bson:"max_score_by_teacher"`
}

func GetAssignment(subjectId, id string) (*Assignment, error) {
	assignment := Assignment{}
	c := mongo.DB("lxchecker").C("assignments")
	if err := c.Find(bson.M{"subject_id": subjectId, "id": id}).One(&assignment); err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		panic(err)
	}
	return &assignment, nil
}

func GetAssignmentOrPanic(subjectId, id string) *Assignment {
	assignment, err := GetAssignment(subjectId, id)
	if err != nil {
		panic(err)
	}
	return assignment
}

func GetAllAssignments(subjectId string) []Assignment {
	assignments := []Assignment{}
	c := mongo.DB("lxchecker").C("assignments")
	if err := c.Find(bson.M{"subject_id": subjectId}).All(&assignments); err != nil {
		panic(err)
	}
	return assignments
}

func InsertAssignment(a Assignment) error {
	if _, err := GetSubject(a.SubjectId); err != nil {
		if err == ErrNotFound {
			return ErrNotFound
		}
		panic(err)
	}
	c := mongo.DB("lxchecker").C("assignments")
	if err := c.Insert(a); err != nil {
		if mgo.IsDup(err) {
			return ErrAlreadyExists
		}
		panic(err)
	}
	return nil
}
