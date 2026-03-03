#!/usr/bin/env bash
set -euo pipefail

module_path="$(go list -m)"

domain_forbidden_pattern="^${module_path}/internal/(adapter|app)(/|$)"
app_forbidden_pattern="^${module_path}/internal/adapter/discord(/|$)"

failures=0

while IFS= read -r pkg; do
  imports="$(go list -f '{{join .Imports "\n"}}' "${pkg}")"

  if [[ "${pkg}" == "${module_path}/internal/domain" || "${pkg}" == "${module_path}/internal/domain/"* ]]; then
    if echo "${imports}" | rg -n "${domain_forbidden_pattern}" >/dev/null 2>&1; then
      echo "Dependency rule violation:"
      echo "  package: ${pkg}"
      echo "  rule: domain must not import internal/adapter or internal/app"
      echo "  matched imports:"
      echo "${imports}" | rg -n "${domain_forbidden_pattern}" || true
      failures=$((failures + 1))
    fi
  fi

  if [[ "${pkg}" == "${module_path}/internal/app" || "${pkg}" == "${module_path}/internal/app/"* ]]; then
    if echo "${imports}" | rg -n "${app_forbidden_pattern}" >/dev/null 2>&1; then
      echo "Dependency rule violation:"
      echo "  package: ${pkg}"
      echo "  rule: app must not import internal/adapter/discord"
      echo "  matched imports:"
      echo "${imports}" | rg -n "${app_forbidden_pattern}" || true
      failures=$((failures + 1))
    fi
  fi
done < <(go list ./...)

if [[ "${failures}" -gt 0 ]]; then
  echo "Dependency direction check failed with ${failures} violation(s)."
  exit 1
fi

echo "Dependency direction check passed."
