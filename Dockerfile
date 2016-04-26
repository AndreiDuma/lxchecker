FROM ubuntu:14.04
MAINTAINER Andrei Duma

# add the server
ADD https://github.com/AndreiDuma/lxchecker/releases/download/v0.1/scheduler /srv/scheduler
RUN chmod +x /srv/scheduler
EXPOSE 8080

CMD /srv/scheduler