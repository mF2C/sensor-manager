#!/bin/sh

# copies x86 and arm images from one registry to another, not including the manifests

if [ -z $1 ]; then
    echo "Missing first parameter: no source image specified."
    exit 1
fi

if [ -z $2 ]; then
    echo "Missing second parameter: no destination image specified."
    exit 2
fi

set -e
set -u

SOURCE_IMAGE="$1"
DESTINATION_IMAGE="$2"

echo "Publishing $SOURCE_IMAGE into $DESTINATION_IMAGE"
echo "    pulling x86_64 source"
docker pull ${SOURCE_IMAGE}:latest-x86_64
echo "    pulling armhf source"
docker pull ${SOURCE_IMAGE}:latest-armhf
echo "    re-tagging x86_64"
docker tag ${SOURCE_IMAGE}:latest-x86_64 ${DESTINATION_IMAGE}:latest-x86_64
echo "    re-tagging armhf"
docker tag ${SOURCE_IMAGE}:latest-armhf ${DESTINATION_IMAGE}:latest-armhf
echo "    pushing x86_64"
docker push ${DESTINATION_IMAGE}:latest-x86_64
echo "    pushing armhf"
docker push ${DESTINATION_IMAGE}:latest-armhf
echo "    done!"
