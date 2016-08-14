# lxchecker
Assignment checking using Linux Containers

## how to deploy
1. run Docker
2. run MongoDB
3. run frontend

## TODO
* remove admin
* remove teacher
* delete user (+submissions, +teacher[s])
* remove hidden fields in forms (subject.html)
* log changes (with user) like grading (for responsibility)

* rename Teacher to TeacherRole?
* move form parsing to GetRequestData?

## done
* use RequestData in assignments.go
* remove usage of CurrentUser
* create RequireAdmin middleware for POSTs
* create RequireTeacher middleware for POSTs
* remove mgo references outside of package db (like mgo.ErrNotFound)
