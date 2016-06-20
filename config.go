package config

import "github.com/gorilla/sessions"

var (
	// TODO: make this key private.
	CookieStore = sessions.NewCookieStore([]byte("super-secret-key"))

	// TODO: add a global logger.
)
