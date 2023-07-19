#!/bin/bash

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
