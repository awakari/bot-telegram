#!/bin/bash

export SLUG=ghcr.io/awakari/bot-telegram
export VERSION=latest
docker tag awakari/bot-telegram "${SLUG}":"${VERSION}"
docker push "${SLUG}":"${VERSION}"
