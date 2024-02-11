FROM ubuntu:14.04
MAINTAINER Andrei Duma

# install required packages
RUN apt-get update
RUN apt-get install -y build-essential
RUN apt-get install -y unzip

# create folder for submission
RUN mkdir /submission/

# add checker code
ADD checker/ /checker/

# run tests when container is started
CMD /checker/check.sh

# keep the container awake for debugging purposes
#CMD sleep 1000
