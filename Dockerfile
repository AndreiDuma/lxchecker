FROM ubuntu:14.04
MAINTAINER Andrei Duma

RUN apt-get update
RUN apt-get install unzip

# install lxchecker
WORKDIR /srv/
ADD https://github.com/AndreiDuma/lxchecker/releases/download/v0.1alpha/release.zip release.zip
RUN unzip release.zip
RUN rm release.zip
RUN chmod +x lxchecker
EXPOSE 8080

CMD ./lxchecker
