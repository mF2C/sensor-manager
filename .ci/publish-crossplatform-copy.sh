#!/bin/sh

# copies x86 and arm images from one registry to another, not including the manifests

if [ -z $1 ]; then
    echo "Missing first parameter: no source image specified."
    exit 1
fi

if [ -z $2 ]; then
    echo "Missing second parameter: no source tag base specified."
    exit 2
fi

if [ -z $3 ]; then
    echo "Missing third parameter: no destination image specified."
    exit 3
fi

if [ -z $4 ]; then
    echo "Missing fourth parameter: no destination tag base specified."
    exit 4
fi

set -e
set -u

SOURCE_IMAGE="$1"
SOURCE_TAG="$2"
DESTINATION_IMAGE="$3"
DESTINATION_TAG="$4"

echo "Publishing $SOURCE_IMAGE:$SOURCE_TAG-* into $DESTINATION_IMAGE:$DESTINATION_TAG-*"
echo "    pulling x86_64 source"
docker pull ${SOURCE_IMAGE}:${SOURCE_TAG}-x86_64
echo "    pulling armhf source"
docker pull ${SOURCE_IMAGE}:${SOURCE_TAG}-armhf
echo "    re-tagging x86_64"
docker tag ${SOURCE_IMAGE}:${SOURCE_TAG}-x86_64 ${DESTINATION_IMAGE}:${DESTINATION_TAG}-x86_64
echo "    re-tagging armhf"
docker tag ${SOURCE_IMAGE}:${SOURCE_TAG}-armhf ${DESTINATION_IMAGE}:${DESTINATION_TAG}-armhf
echo "    pushing x86_64"
docker push ${DESTINATION_IMAGE}:${DESTINATION_TAG}-x86_64
echo "    pushing armhf"
docker push ${DESTINATION_IMAGE}:${DESTINATION_TAG}-armhf
echo "    done!"
