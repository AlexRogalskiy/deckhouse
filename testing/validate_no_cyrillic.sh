#!/bin/bash

# Copyright 2021 Flant JSC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

function request_gitlab_api() {
  curl --silent -f -H "PRIVATE-TOKEN: ${JOB_TOKEN}"  https://fox.flant.com/api/v4/projects/${PROJECT_ID}/${1}
}

function check_cyrillic_letters() {
  if words=$(grep -Eo "[А-Яа-яЁё]+" <<< ${1}) ; then
    echo "  ERROR: Cyrillic letters found!"
    echo "${words}" | jq -R '.' | jq -sc '.'
    return 1
  fi
  echo "  OK!"
}

function main() {
  MERGE_REQUEST_ID=$(request_gitlab_api "merge_requests?state=opened" | jq -r --arg c ${COMMIT_SHA} '.[]|select(.sha == $c) | .iid')
  if [[ "$MERGE_REQUEST_ID" == "" ]]; then
    echo "No merge request found for commit sha: ${COMMIT_SHA}"
    exit 0
  fi

  echo "Merge Request ID = ${MERGE_REQUEST_ID}"

  MERGE_REQUEST_DATA=$(request_gitlab_api "merge_requests/${MERGE_REQUEST_ID}/changes" | jq -r '.')
  if [[ $(jq -rc '.labels[] | select ( . == "Content: Cyrillic")' <<< ${MERGE_REQUEST_DATA}) != "" ]]; then
    echo '"Content: Cyrillic" label is present. End.'
    exit 0
  fi

  hasCyrillicLetters=0

  echo -n "Check title:"
  if ! check_cyrillic_letters "$(jq -r '.title' <<< ${MERGE_REQUEST_DATA})" ; then
    hasCyrillicLetters=1
  fi

  echo -n "Check description:"
  if ! check_cyrillic_letters "$(jq -r '.description' <<< ${MERGE_REQUEST_DATA})" ; then
    hasCyrillicLetters=1
  fi

  echo "Check diff:"
  CHANGES=$(jq -rc '.changes' <<< ${MERGE_REQUEST_DATA})

  for key in $(jq -rc 'keys[]' <<< "${CHANGES}"); do
    FILE_DIFF=$(jq -rc --arg key "$key" '.[$key | tonumber]' <<< "${CHANGES}")

    new_path=$(jq -rc '.new_path' <<< ${FILE_DIFF})

    # skip documentation
    pattern="doc-ru-.+.y[a]?ml$|_RU.md$|_ru.html$|docs/site/_.+|docs/documentation/_.+"
    if [[ "$new_path" =~ $pattern ]] ; then
      echo "  * skip documentation: $new_path"
      continue
    fi

    # skip translations
    if [[ "$new_path" == *"/i18n/"* ]] ; then
      echo "  * skip translations: $new_path"
      continue
    fi

    # skip self
    if [[ "$new_path" == *"validate_no_cyrillic.sh" ]] ; then
      echo "  * skip self: $new_path"
      continue
    fi

    # Check only new or updated lines in diff
    # Ignore latin only lines.
    if ! diff_error=$(check_cyrillic_letters "$(jq -rc '.diff' <<< ${FILE_DIFF} | grep '+')" ) ; then
      echo "  * diff: ${new_path} ${diff_error}"
      hasCyrillicLetters=1
    fi
  done

  exit $hasCyrillicLetters
}

PROJECT_ID=$1
COMMIT_SHA=$2
JOB_TOKEN=$3

main
