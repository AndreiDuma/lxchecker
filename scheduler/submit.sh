#!/bin/bash

curl -X POST -F submission=@"$1" localhost:8080/submit
