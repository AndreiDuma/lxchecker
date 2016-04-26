#!/bin/bash

curl -X POST -F submission=@"$1" http://localhost:8080/submit
