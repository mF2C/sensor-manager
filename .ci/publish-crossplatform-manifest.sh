#!/bin/sh

# creates a cross-platform manifest for x86 and arm

if [ -z $1 ]; then
    echo "Missing first parameter: no base image specified."
    exit 1
fi

if [ -z $2 ]; then
    echo "Missing second parameter: no image tag specified."
    exit 2
fi

set -e
set -u

BASE_IMAGE="$1"
BASE_IMAGE_TAG="$2"

echo "Creating manifest: $BASE_IMAGE:$BASE_IMAGE_TAG = $BASE_IMAGE:$BASE_IMAGE_TAG-x86_64 + $BASE_IMAGE:$BASE_IMAGE_TAG-armhf"
docker manifest create ${BASE_IMAGE}:${BASE_IMAGE_TAG} ${BASE_IMAGE}:${BASE_IMAGE_TAG}-x86_64 ${BASE_IMAGE}:${BASE_IMAGE_TAG}-armhf
echo "    annotating x86_64 manifest"
docker manifest annotate ${BASE_IMAGE}:${BASE_IMAGE_TAG} ${BASE_IMAGE}:${BASE_IMAGE_TAG}-x86_64 --arch amd64
echo "    annotating armhf manifest"
docker manifest annotate ${BASE_IMAGE}:${BASE_IMAGE_TAG} ${BASE_IMAGE}:${BASE_IMAGE_TAG}-armhf --arch arm
echo "    pushing manifest"
docker manifest push ${BASE_IMAGE}:${BASE_IMAGE_TAG}
echo "    done!"
