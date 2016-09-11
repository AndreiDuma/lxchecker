build:
	go build

zip: build
	cp -r web/templates/ .
	cp -r web/static/ .

	zip -r release.zip lxchecker templates/ static/

	rm -r templates/ static/

image: build
	docker build -t lxchecker/lxchecker .

push: image
	docker push lxchecker/lxchecker

clean:
	rm lxchecker
