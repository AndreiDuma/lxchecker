package main

import (
	"html/template"
	"net/http"

	"github.com/AndreiDuma/lxchecker"
	"github.com/AndreiDuma/lxchecker/db"
)

var (
	loginTmpl  = template.Must(template.ParseFiles("templates/base.html", "templates/login.html"))
	signupTmpl = template.Must(template.ParseFiles("templates/base.html", "templates/signup.html"))
)

// LoginTmplHandler serves the login page.
func LoginTmplHandler(w http.ResponseWriter, r *http.Request) {
	continueURL := r.FormValue("continue")
	signupURL := "/signup"
	if continueURL != "" {
		signupURL = "/signup?continue=" + continueURL
	}
	loginTmpl.Execute(w, struct {
		ContinueURL string
		SignupURL   string
	}{
		continueURL,
		signupURL,
	})
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
	http.Redirect(w, r, "/login", http.StatusFound)
}

// SignupTmplHandler serves the sign up page.
func SignupTmplHandler(w http.ResponseWriter, r *http.Request) {
	continueURL := r.FormValue("continue")
	loginURL := "/login"
	if continueURL != "" {
		loginURL = "/login?continue=" + continueURL
	}
	signupTmpl.Execute(w, struct {
		ContinueURL string
		LoginURL    string
	}{
		continueURL,
		loginURL,
	})
}

// SignupHandler creates a new account.
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	retrySignup := func() {
		signupURL := "/signup?error=true"
		continueURL := r.FormValue("continue")
		if continueURL != "" {
			signupURL += "&continue=" + continueURL
		}
		http.Redirect(w, r, signupURL, http.StatusFound)
	}

	u := &db.User{}

	// Get username from request.
	u.Username = r.FormValue("username")
	if u.Username == "" {
		retrySignup()
		return
	}

	// Get password from request.
	// TODO: use password hash instead of plain-text password.
	u.Password = r.FormValue("password")
	if u.Password == "" {
		retrySignup()
		return
	}

	// Insert user.
	if err := db.InsertUser(u); err != nil {
		if err == db.ErrAlreadyExists {
			retrySignup()
			return
		}
		panic(err)
	}

	// Set auth cookie.
	session, _ := config.CookieStore.Get(r, "auth")
	session.Values["username"] = u.Username
	session.Save(r, w)

	// Redirect to `continueURL`.
	continueURL := r.FormValue("continue")
	if continueURL != "" {
		http.Redirect(w, r, continueURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/-/", http.StatusFound)
}
