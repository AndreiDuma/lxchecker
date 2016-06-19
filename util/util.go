package util

import (
	"log"
	"net/http"

	"github.com/AndreiDuma/lxchecker"
	"github.com/AndreiDuma/lxchecker/db"
	"github.com/gorilla/context"
)

func LogPanics() {
	log.Printf("panic: %v\n", recover())
}

// CurrentUser returns the currently logged in user or nil if there is none.
func CurrentUser(r *http.Request) *db.User {
	user, _ := context.Get(r, "user").(*db.User)
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

		session, err := config.CookieStore.Get(r, "auth")
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
