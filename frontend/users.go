package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// TODO: make this key private.
	store = sessions.NewCookieStore([]byte("super-secret-key"))

	loginTmpl = template.Must(template.ParseFiles("templates/login.html"))
)

// User holds data related to an user.
type User struct {
	Username string
	Password string
}

// GetUser returns the currently logged in user or nil if there is none.
func GetUser(r *http.Request) *User {
	user, _ := context.Get(r, "user").(*User)
	return user
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

		session, err := store.Get(r, "auth")
		if err != nil {
			requireLogin()
			return
		}

		username := session.Values["username"]
		if username == "" {
			requireLogin()
			return
		}

		user := User{}
		c := mongo.DB("lxchecker").C("users")
		if err := c.Find(bson.M{"username": username}).One(&user); err != nil {
			// TODO: consider also treating DB errors here.
			requireLogin()
			return
		}
		context.Set(r, "user", &user)

		next.ServeHTTP(w, r)
	})
}

// LoginTmplHandler serves the login pages.
func LoginTmplHandler(w http.ResponseWriter, r *http.Request) {
	continueURL := r.FormValue("continue")
	loginTmpl.Execute(w, struct {
		ContinueURL string
	}{continueURL})
}

// LoginHandler checkes the user's credentials and creates a session.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	retryLogin := func() {
		loginURL := "/login?error=true"
		continueURL := r.FormValue("continue")
		if continueURL != "" {
			loginURL += "&continue=" + continueURL
		}
		http.Redirect(w, r, loginURL, http.StatusFound)
	}

	// Get username from request.
	username := r.FormValue("username")
	if username == "" {
		retryLogin()
		return
	}

	// Get password from request.
	// TODO: use password hash instead of plain-text password.
	password := r.FormValue("password")
	if password == "" {
		retryLogin()
		return
	}

	user := User{}
	c := mongo.DB("lxchecker").C("users")
	if err := c.Find(bson.M{"username": username, "password": password}).One(&user); err != nil {
		if err == mgo.ErrNotFound {
			retryLogin()
			return
		}
		log.Println("error during authentication:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "auth")
	session.Values["username"] = username
	session.Save(r, w)

	// Redirect to `continueURL`.
	continueURL := r.FormValue("continue")
	if continueURL != "" {
		http.Redirect(w, r, continueURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/-/", http.StatusFound)
}

// LogoutHandler delete's a user's session, logging them out.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}
