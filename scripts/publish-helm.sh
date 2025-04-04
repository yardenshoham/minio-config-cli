#!/bin/bash

set -ex

# This script is used to publish the helm chart to the yardenshohamcharts repo
# It requires the following environment variables to be set:
#   - DOCKERHUB_CHARTS_USERNAME
#   - DOCKERHUB_CHARTS_TOKEN

if [ -z "$DOCKERHUB_CHARTS_USERNAME" ]; then
  echo "DOCKERHUB_CHARTS_USERNAME is not set"
  exit 1
fi

if [ -z "$DOCKERHUB_CHARTS_TOKEN" ]; then
  echo "DOCKERHUB_CHARTS_TOKEN is not set"
  exit 1
fi

if [ -z "$1" ]; then
  echo "Version is not set"
  exit 1
fi

# Remove 'v' prefix if present
VERSION=${1#v}

echo "Logging in to DockerHub"
helm registry login registry-1.docker.io -u $DOCKERHUB_CHARTS_USERNAME -p $DOCKERHUB_CHARTS_TOKEN

echo "Packaging helm chart"
helm package chart --version $VERSION --app-version $VERSION --dependency-update

echo "Pushing helm chart to yardenshohamcharts repo"
helm push minio-config-cli-$VERSION.tgz oci://registry-1.docker.io/yardenshohamcharts