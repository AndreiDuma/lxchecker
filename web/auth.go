package main

import (
	"html/template"
	"net/http"

	"github.com/AndreiDuma/lxchecker"
	"github.com/AndreiDuma/lxchecker/db"
)

var (
	loginTmpl = template.Must(template.ParseFiles("templates/login.html"))
)

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

	// Check user credentials.
	if _, err := db.GetUserAuth(username, password); err != nil {
		if err == db.ErrNotFound {
			retryLogin()
			return
		}
		panic(err)
	}

	// Set auth cookie.
	session, _ := config.CookieStore.Get(r, "auth")
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
	session, _ := config.CookieStore.Get(r, "auth")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}
