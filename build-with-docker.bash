#!/bin/bash

docker run --rm -v ./:/data -w /data golang:latest go build -v -buildvcs=false
