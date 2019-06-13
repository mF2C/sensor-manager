#!/bin/sh

# creates a cross-platform manifest for x86 and arm

if [ -z $1 ]; then
    echo "Missing first parameter: no base image specified."
    exit 1
fi

set -e
set -u

BASE_IMAGE="$1"

echo "Creating manifest: $BASE_IMAGE:latest = $BASE_IMAGE:latest-x86_64 + $BASE_IMAGE:latest-armhf"
docker manifest create ${BASE_IMAGE}:latest ${BASE_IMAGE}:latest-x86_64 ${BASE_IMAGE}:latest-armhf
echo "    annotating x86_64 manifest"
docker manifest annotate ${BASE_IMAGE}:latest ${BASE_IMAGE}:latest-x86_64 --arch amd64
echo "    annotating armhf manifest"
docker manifest annotate ${BASE_IMAGE}:latest ${BASE_IMAGE}:latest-armhf --arch arm
echo "    pushing manifest"
docker manifest push ${BASE_IMAGE}:latest
echo "    done!"
