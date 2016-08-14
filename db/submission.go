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
	Metadata         map[string]string

	GradedByTeacher bool   `bson:"graded_by_teacher"`
	ScoreByTests    uint64 `bson:"score_by_tests"`
	ScoreByTeacher  uint64 `bson:"score_by_teacher"`
	Feedback        string
	GraderUsername  string `bson:"grader_username"`

	Overdue bool
	Penalty uint64
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

func GetSubmissionOrPanic(subjectId, assignmentId, id string) *Submission {
	submission, err := GetSubmission(subjectId, assignmentId, id)
	if err != nil {
		panic(err)
	}
	return submission
}

func GetAllSubmissions(subjectId, assignmentId string) []Submission {
	submissions := []Submission{}
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Find(bson.M{
		"subject_id":    subjectId,
		"assignment_id": assignmentId,
	}).Sort("-timestamp").All(&submissions); err != nil {
		panic(err)
	}
	return submissions
}

func GetSubmissionsOfUser(subjectId, assignmentId, ownerUsername string) []Submission {
	submissions := []Submission{}
	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Find(bson.M{
		"subject_id":     subjectId,
		"assignment_id":  assignmentId,
		"owner_username": ownerUsername,
	}).Sort("-timestamp").All(&submissions); err != nil {
		panic(err)
	}
	return submissions
}

func GetActiveSubmissions(subjectId, assignmentId string) []Submission {
	type ActiveSubmission struct {
		OwnerUsername string `bson:_id`
		Submission    Submission
	}
	submissionsByUser := []ActiveSubmission{}

	c := mongo.DB("lxchecker").C("submissions")
	if err := c.Pipe([]bson.M{
		{
			"$match": bson.M{
				"subject_id":    subjectId,
				"assignment_id": assignmentId,
			},
		},
		{
			"$sort": bson.M{
				"timestamp": -1,
			},
		},
		{
			"$group": bson.M{
				"_id":        "$owner_username",
				"submission": bson.M{"$first": "$$ROOT"},
			},
		},
		{
			"$sort": bson.M{
				"submission.timestamp": -1,
			},
		},
	}).All(&submissionsByUser); err != nil {
		panic(err)
	}

	submissions := []Submission{}
	for _, s := range submissionsByUser {
		submissions = append(submissions, s.Submission)
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
