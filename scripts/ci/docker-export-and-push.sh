#!/bin/sh
# Tag and push a locally built image to GitLab Container Registry and Docker Hub.
set -eu

LOCAL_TAG="${1:?local image tag}"
REGISTRY_IMAGE="${CI_REGISTRY_IMAGE:-${DOCKER_REGISTRY_IMAGE:-registry.gitlab.com/austin-millan/otaclock}}"
DOCKERHUB_IMAGE="${DOCKERHUB_IMAGE:-aamillan/otaclock}"

if [ "${DOCKER_PUSH:-true}" != "true" ]; then
  echo "DOCKER_PUSH is not true - skipping registry push." >&2
  exit 1
fi

mkdir -p dist
: > dist/docker-push-tags.txt
: > dist/dockerhub-push-tags.txt

gitlab_seen=""
gitlab_push() {
  tag="$1"
  [ -z "$tag" ] && return 0
  case " $gitlab_seen " in
    *" $tag "*) return 0 ;;
  esac
  gitlab_seen="${gitlab_seen} ${tag}"
  remote="${REGISTRY_IMAGE}:${tag}"
  docker tag "${LOCAL_TAG}" "${remote}"
  docker push "${remote}"
  echo "Pushed ${remote}"
  echo "${remote}" >> dist/docker-push-tags.txt
}

dockerhub_seen=""
dockerhub_push() {
  tag="$1"
  [ -z "$tag" ] && return 0
  case " $dockerhub_seen " in
    *" $tag "*) return 0 ;;
  esac
  dockerhub_seen="${dockerhub_seen} ${tag}"
  remote="${DOCKERHUB_IMAGE}:${tag}"
  docker tag "${LOCAL_TAG}" "${remote}"
  docker push "${remote}"
  echo "Pushed ${remote}"
  echo "${remote}" >> dist/dockerhub-push-tags.txt
}

push_gitlab_tags() {
  if [ -n "${CI_COMMIT_TAG:-}" ]; then
    gitlab_push "${CI_COMMIT_TAG}"
  fi
  gitlab_push "${CI_COMMIT_REF_SLUG}"
  gitlab_push "${CI_COMMIT_SHORT_SHA}"
  if [ "${CI_COMMIT_BRANCH:-}" = "${CI_DEFAULT_BRANCH}" ]; then
    gitlab_push "latest"
  fi
  # Monday scheduled publish (SCHEDULE_KIND=docker): keep short SHA + latest, add weekly.
  if [ "${SCHEDULE_KIND:-}" = "docker" ] && [ "${CI_COMMIT_BRANCH:-}" = "${CI_DEFAULT_BRANCH}" ]; then
    gitlab_push "weekly"
  fi
}

# Docker Hub: release tags, plus latest + master on the default branch.
push_dockerhub_tags() {
  if [ -n "${CI_COMMIT_TAG:-}" ]; then
    dockerhub_push "${CI_COMMIT_TAG}"
    dockerhub_push "latest"
  fi
  if [ "${CI_COMMIT_BRANCH:-}" = "${CI_DEFAULT_BRANCH}" ]; then
    dockerhub_push "latest"
    dockerhub_push "master"
  fi
  if [ "${SCHEDULE_KIND:-}" = "docker" ] && [ "${CI_COMMIT_BRANCH:-}" = "${CI_DEFAULT_BRANCH}" ]; then
    dockerhub_push "weekly"
  fi
}

if [ -z "${CI_REGISTRY:-}" ] || [ -z "${CI_REGISTRY_USER:-}" ] || [ -z "${CI_REGISTRY_PASSWORD:-}" ]; then
  echo "GitLab registry credentials missing. Enable Container Registry for the project." >&2
  exit 1
fi

echo "${CI_REGISTRY_PASSWORD}" | docker login -u "${CI_REGISTRY_USER}" --password-stdin "${CI_REGISTRY}"
push_gitlab_tags

if [ -n "${DOCKERHUB_USER:-}" ] && [ -n "${DOCKERHUB_SECRET:-}" ]; then
  echo "${DOCKERHUB_SECRET}" | docker login -u "${DOCKERHUB_USER}" --password-stdin
  push_dockerhub_tags
else
  echo "DOCKERHUB_USER or DOCKERHUB_SECRET not set - skipping Docker Hub push." >&2
fi

docker image inspect "${LOCAL_TAG}" > dist/docker-image-info.txt

if [ "${CI_COMMIT_BRANCH:-}" = "${CI_DEFAULT_BRANCH}" ] || [ -n "${CI_COMMIT_TAG:-}" ]; then
  version="${CI_COMMIT_TAG:-${CI_COMMIT_SHORT_SHA}}"
  docker save "${LOCAL_TAG}" -o "dist/otaclock2-docker-${version}.tar"
  echo "Saved dist/otaclock2-docker-${version}.tar"
fi
