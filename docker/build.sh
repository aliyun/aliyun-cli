#!/usr/bin/env sh

function main () {
  local CONTAINER_NAME="ellerbrock/alpine-aliyuncli"

  docker build -t ${CONTAINER_NAME} .
}

main

