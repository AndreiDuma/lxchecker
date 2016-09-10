release:
	go build

	cp -r web/templates/ .
	cp -r web/static/ .

	zip -r release.zip lxchecker templates/ static/

	rm -r lxchecker templates/ static/

