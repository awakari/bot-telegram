#!/bin/bash

export SLUG=ghcr.io/awakari/bot-telegram
export VERSION=latest
docker tag awakari/api "${SLUG}":"${VERSION}"
docker push "${SLUG}":"${VERSION}"
