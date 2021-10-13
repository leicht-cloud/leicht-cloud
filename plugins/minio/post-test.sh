#!/bin/sh

# This script is ran by the plugins test suite after running the minio tests
# and cleans up all the stuff that was setup in the pre-test.sh script

docker rm -f minio_test