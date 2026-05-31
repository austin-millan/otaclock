#!/usr/bin/env bash
# GL/X11 development libraries required to build the Ebitengine GUI on Linux.
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y --no-install-recommends \
  pkg-config \
  libgl1-mesa-dev \
  libxrandr-dev \
  libxcursor-dev \
  libxinerama-dev \
  libxi-dev \
  libxxf86vm-dev \
  libasound2-dev \
  ca-certificates \
  git

rm -rf /var/lib/apt/lists/*
