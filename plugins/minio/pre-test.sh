#!/bin/sh

# This script is ran by the plugins test suite before running the minio tests

if ! [ -x "$(command -v docker)" ]; then
    echo "docker could not be found, skipping"
    exit 1
fi

if ! [ -x "$(command -v curl)" ]; then
    echo "curl command could not be found, skipping"
    exit 1
fi

if [ -z $TMPDIR ]; then
    echo "no TMPDIR set, skipping.."
    exit 1
fi

docker rm -f minio_test

docker run -d \
    --name minio_test \
    -p 9000:9000 \
    -e "MINIO_ACCESS_KEY=go-cloud" \
    -e "MINIO_SECRET_KEY=ThisIsASecret" \
    -v "$TMPDIR:/data" \
    -u "$(id -u):$(id -g)" \
    minio/minio:latest server /data

while ! curl http://127.0.0.1:9000/ > /dev/null 2>&1; do
  sleep 1
done
