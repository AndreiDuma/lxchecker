package util

import (
	"fmt"
	"log"
	"net/http"

	"github.com/AndreiDuma/lxchecker"
	"github.com/AndreiDuma/lxchecker/db"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var (
	// TODO: make this key private.
	cookieStore = sessions.NewCookieStore([]byte("super-secret-key"))
)

type RequestData struct {
	SubjectId    string
	AssignmentId string
	SubmissionId string

	User           *db.User
	UserIsLoggedIn bool
	UserIsTeacher  bool
	UserIsAdmin    bool
}

func GetRequestData(r *http.Request) *RequestData {
	rd := &RequestData{}

	vars := mux.Vars(r)
	rd.SubjectId = vars["subject_id"]
	rd.AssignmentId = vars["assignment_id"]
	rd.SubmissionId = vars["submission_id"]

	rd.User, _ = context.Get(r, "user").(*db.User)
	if rd.User != nil {
		// For convenience.
		rd.UserIsLoggedIn = true
		rd.UserIsAdmin = rd.User.IsAdmin

		// If within a subject, also determine if the user is teacher for that subject.
		if rd.SubjectId != "" {
			rd.UserIsTeacher = db.IsTeacher(rd.User.Username, rd.SubjectId)
		}
	}
	return rd
}

// RequireAuth middleware makes sure users are logged in by first redirecting
// them to the login page.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireLogin := func() {
			// Save requested URL in query param and redirect to login.
			loginURL := "/login?continue=" + r.URL.Path
			http.Redirect(w, r, loginURL, http.StatusFound)
		}

		session, err := cookieStore.Get(r, "auth")
		if err != nil {
			requireLogin()
			return
		}

		username, ok := session.Values["username"].(string)
		if !ok || username == "" {
			requireLogin()
			return
		}

		user, err := db.GetUser(username)
		if err != nil {
			if err == db.ErrNotFound {
				requireLogin()
				return
			}
			panic(err)
		}
		context.Set(r, "user", user)

		next.ServeHTTP(w, r)
	})
}

// RequireAdmin middleware makes sure the current user is an admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rd := GetRequestData(r)

		if !rd.UserIsAdmin {
			// TODO: call a proper handler or redirect.
			fmt.Fprintf(w, "permission denied: need to be admin")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireTeacherOrAdmin middleware makes sure the current user is a teacher for the requested subject or an admin.
func RequireTeacherOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rd := GetRequestData(r)

		if !rd.UserIsTeacher && !rd.UserIsAdmin {
			// TODO: call a proper handler or redirect.
			fmt.Fprintf(w, "permission denied: need to be teacher or admin")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func LogPanics() {
	r := recover()
	if r != nil {
		log.Printf("panic: %v\n", r)
	}
}
