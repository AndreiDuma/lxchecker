# Lxchecker
Assignment checking using Linux Containers

## How to install Lxchecker

### MongoDB

    $ docker run -d --name db mongo

### Docker Swarm

    TODO

### Lxchecker

    $ docker run -d --name lxchecker --link db -p 80:8080 lxchecker/lxchecker
