#!/bin/bash

curl -X POST -F submission=@"$1" http://lxchecker.andreiduma.ro/submit
