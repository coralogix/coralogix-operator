#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
chart=charts/coralogix-operator
app_version=$(awk '/^appVersion:/ { print $2 }' "$chart/Chart.yaml")

assert_image() {
  local expected=$1
  shift
  local rendered actual
  rendered=$(helm template test "$chart" --show-only templates/deployment.yaml \
    --set secret.data.apiKey=test-key --set coralogixOperator.domain=gov.example.com "$@")
  actual=$(awk '/^[[:space:]]*image:/ { print $2 }' <<< "$rendered")
  if [[ "$actual" != "$expected" ]]; then
    echo "Expected image $expected, got $actual" >&2
    exit 1
  fi
}

assert_image "coralogixrepo/coralogix-operator:v$app_version"
assert_image "coralogixrepo/coralogix-operator:v$app_version-fips" \
  --set coralogixOperator.image.fips=true
assert_image "coralogixrepo/coralogix-operator:v1.2.3" \
  --set-string coralogixOperator.image.tag=1.2.3
assert_image "coralogixrepo/coralogix-operator:v1.2.3-fips" \
  --set-string coralogixOperator.image.tag=1.2.3 --set coralogixOperator.image.fips=true
assert_image "coralogixrepo/coralogix-operator:v1.2.3-fips" \
  --set-string coralogixOperator.image.tag=1.2.3-fips --set coralogixOperator.image.fips=true
assert_image "coralogixrepo/coralogix-operator:v1.2.3-fips" \
  --set-string coralogixOperator.image.tag=1.2.3-fips
assert_image "registry.example.com/operator:v1.2.3-rc.1-fips" \
  --set-string coralogixOperator.image.repository=registry.example.com/operator \
  --set-string coralogixOperator.image.tag=1.2.3-rc.1 --set coralogixOperator.image.fips=true

echo "Helm image selection checks passed"
