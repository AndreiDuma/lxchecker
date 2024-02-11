#!/bin/bash

cd /submission
unzip submission.zip
make
cp /submission/libvmsim.so /checker
cd /checker
make -f Makefile.checker
