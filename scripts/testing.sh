#!/bin/bash

export SLUG=ghcr.io/awakari/bot-telegram
export VERSION=$(git rev-parse --short HEAD)
docker tag awakari/bot-telegram "${SLUG}":"${VERSION}"
docker push "${SLUG}":"${VERSION}"
