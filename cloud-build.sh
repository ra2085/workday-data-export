#!/bin/bash

set -e

export PROJECT_ID="$(gcloud config get project)"

if [ -z "${PROJECT_ID}" ] ; then
  echo "error: PROJECT_ID must be set in gcloud"
fi

gcloud services enable \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com

gcloud builds submit . --tag "gcr.io/${PROJECT_ID}/workday-entity-export"