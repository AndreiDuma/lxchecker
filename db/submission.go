package db

import (
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Submission holds data related to a submission.
type Submission struct {
	Id           string
	AssignmentId string `bson:"assignment_id"`
	SubjectId    string `bson:"subject_id"`

	OwnerUsername string `bson:"owner_username"`

	Status           string // TODO: make this a constant or an enum.
	Timestamp        time.Time
	UploadedFile     []byte `bson:"uploaded_file",json:"-"`
	UploadedFileName string `bson:"uploaded_file_name",json:"-"`
	Logs             []byte
	Score            uint
	Feedback         string
}

func GetSubmission(subjectId, assignmentId, id string) (*Submission, error) {
	submission := Submission{}
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Find(bson.M{
		"subject_id":    subjectId,
		"assignment_id": assignmentId,
		"id":            id,
	}).One(&submission); err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		panic(err)
	}
	return &submission, nil
}

func GetAllSubmissions(subjectId, assignmentId string) []Submission {
	submissions := []Submission{}
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Find(bson.M{
		"subject_id":    subjectId,
		"assignment_id": assignmentId,
	}).All(&submissions); err != nil {
		panic(err)
	}
	return submissions
}

func GetAllSubmissionsOfUser(subjectId, assignmentId, ownerUsername string) []Submission {
	submissions := []Submission{}
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Find(bson.M{
		"subject_id":     subjectId,
		"assignment_id":  assignmentId,
		"owner_username": ownerUsername,
	}).All(&submissions); err != nil {
		panic(err)
	}
	return submissions
}

func NewSubmissionId() string {
	return bson.NewObjectId().Hex()
}

func InsertSubmission(s *Submission) error {
	if _, err := GetAssignment(s.SubjectId, s.AssignmentId); err != nil {
		if err == ErrNotFound {
			return ErrNotFound
		}
		panic(err)
	}
	if _, err := GetUser(s.OwnerUsername); err != nil {
		if err == ErrNotFound {
			return ErrNotFound
		}
		panic(err)
	}
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Insert(s); err != nil {
		if mgo.IsDup(err) {
			return ErrAlreadyExists
		}
		panic(err)
	}
	return nil
}

func UpdateSubmission(s *Submission) error {
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Update(bson.M{
		"subject_id":    s.SubjectId,
		"assignment_id": s.AssignmentId,
		"id":            s.Id,
	}, s); err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		panic(err)
	}
	return nil
}
