#!/bin/bash

# Copyright 2024 Google LLC All Rights Reserved.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#    http://www.apache.org/licenses/LICENSE-2.0
   
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e
if [ -n "${GITHUB_REF}" ]; then
  tag_name="${GITHUB_REF##*/}"
else
  tag_name=v$(date '+%y%m%d')-dev
fi

VERSION_VAR=github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/messages.Version
TARGETS=(
  windows,amd64,.exe,zip
  windows,arm,.exe,zip
  linux,amd64,,tar.gz
  linux,arm,,tar.gz
  linux,arm64,,tar.gz
  darwin,amd64,,tar.gz
  darwin,arm64,,tar.gz
)

cd "$(dirname "${BASH_SOURCE[0]}")/.."

for target in "${TARGETS[@]}"; do
  IFS=, read os arch suffix package <<<$target
  dir=dist/${os}_${arch}
  GOOS=$os GOARCH=$arch go build \
    -ldflags="-X '${VERSION_VAR}=${tag_name}'" \
    -o "$dir/mc2bq${suffix}"
  pushd $dir
  case "${package}" in
    zip)
      fname=mc2bq_${tag_name}_${os}_${arch}.zip
      zip "${fname}" *
    ;;
    tar*)
      fname=mc2bq_${tag_name}_${os}_${arch}.${package}
      tar -cf "${fname}" *
    ;;
    *)
    exit 1
    ;;
  esac
  popd
done
