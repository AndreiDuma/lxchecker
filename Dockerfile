FROM ubuntu:14.04
MAINTAINER Andrei Duma

# install lxchecker
WORKDIR /srv/
COPY lxchecker lxchecker
COPY web/templates/ templates/
COPY web/static/ static/
RUN chmod +x lxchecker
EXPOSE 8080

CMD ./lxchecker
