package db

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// User holds data related to an user.
type User struct {
	Username string
	Password string
}

func GetUser(username string) (*User, error) {
	u := User{}
	c := mongo.DB("lxchecker").C("users")
	if err := c.Find(bson.M{"username": username}).One(&u); err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		panic(err)
	}
	return &u, nil
}

func GetUserAuth(username, password string) (*User, error) {
	u := User{}
	c := mongo.DB("lxchecker").C("users")
	if err := c.Find(bson.M{
		"username": username,
		"password": password,
	}).One(&u); err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		panic(err)
	}
	return &u, nil
}
