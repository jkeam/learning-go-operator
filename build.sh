#!/bin/bash

NUM=9
make podman-build IMG=quay.io/jkeam/learning-go-operators:v$NUM
make podman-push IMG=quay.io/jkeam/learning-go-operators:v$NUM
make bundle IMG=quay.io/jkeam/learning-go-operators:v$NUM
make bundle-build IMG=quay.io/jkeam/learning-go-operators-bundle:v$NUM
podman tag jonkeam.com/learning-go-operators-bundle:v0.0.1 quay.io/jkeam/learning-go-operators-bundle:v$NUM
podman push quay.io/jkeam/learning-go-operators-bundle:v$NUM
operator-sdk run bundle -n jon quay.io/jkeam/learning-go-operators-bundle:v$NUM
