package db

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// User holds data related to an user.
type User struct {
	Username string
	Password string
	IsAdmin  bool `bson:"is_admin"`
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

func GetAdmins() []User {
	admins := []User{}
	c := mongo.DB("lxchecker").C("users")
	if err := c.Find(bson.M{
		"is_admin": true,
	}).All(&admins); err != nil {
		panic(err)
	}
	return admins
}

func InsertUser(u *User) error {
	c := mongo.DB("lxchecker").C("users")
	if err := c.Insert(u); err != nil {
		if mgo.IsDup(err) {
			return ErrAlreadyExists
		}
		panic(err)
	}
	return nil
}

func UpdateUser(u *User) error {
	c := mongo.DB("lxchecker").C("users")
	if err := c.Update(bson.M{
		"username": u.Username,
	}, u); err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		panic(err)
	}
	return nil
}
