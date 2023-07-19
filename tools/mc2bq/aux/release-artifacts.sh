#!/bin/bash
set -e

cd "$(dirname "${BASH_SOURCE[0]}")/.."

if [ -n "${GITHUB_REF}" ]; then
  tag_name="${GITHUB_REF##*/}"
else
  tag_name=dev
fi

find . -name "mc2bq_${tag_name}_*.tar.gz" -or -name "mc2bq_${tag_name}_*.zip" | while read -r artifact; do
  if [ "${tag_name}" == "dev" ]; then
    echo "Dev version not releasing"
    echo ${artifact}
  else
    hub release create "${artifact}" "${tag_name}" -m "${tag_name}"
  fi
done
