  The ROSA CLI tests are controlled by the E2E_SKIP_ROSA_CLI environment variable:

  E2E_SKIP_ROSA_CLI="${E2E_SKIP_ROSA_CLI:-true}"  # Set to "true" to skip

  Default behavior: ROSA CLI tests are skipped by default (defaults to true).

  ROSA_REPO_URL="${ROSA_REPO_URL:-https://github.com/openshift/rosa}"
  ROSA_REPO_BRANCH="${ROSA_REPO_BRANCH:-hyperfleet-v2}"
  ROSA_LABEL_FILTER="${ROSA_LABEL_FILTER:-}"
  ROSA_TEST_PROFILE="${ROSA_TEST_PROFILE:-rosa-hcp-basic}"

  The tests run when E2E_SKIP_ROSA_CLI is false:

  if [[ "${E2E_SKIP_ROSA_CLI}" == "false" ]] || [[ -z "${E2E_SKIP_ROSA_CLI:-}" ]]; then
    echo ""
    echo "=== ROSA CLI Tests ==="
    echo ""
    export ROSA_REPO_URL ROSA_REPO_BRANCH TEST_PROFILE="${ROSA_TEST_PROFILE}"
    export GOTOOLCHAIN=auto
    ROSA_LABEL_FILTER="${ROSA_LABEL_FILTER}" make test-e2e-rosa-cli || rosa_cli_rc=$?

  All ROSA CLI variables are passed through to the container:

  -e "ROSA_REPO_URL=${ROSA_REPO_URL:-}" \
  -e "ROSA_REPO_BRANCH=${ROSA_REPO_BRANCH:-}" \
  -e "ROSA_LABEL_FILTER=${ROSA_LABEL_FILTER:-}" \
  -e "ROSA_TEST_PROFILE=${ROSA_TEST_PROFILE:-}" \
  -e "E2E_SKIP_ROSA_CLI=${E2E_SKIP_ROSA_CLI:-}" \

  Usage Examples

  Run all e2e tests including ROSA CLI:

  E2E_SKIP_ROSA_CLI=false make ephemeral-e2e ID=<your-env-id>

  Run ROSA CLI tests with a specific label filter:

  E2E_SKIP_ROSA_CLI=false ROSA_LABEL_FILTER="smoke" make ephemeral-e2e ID=<your-env-id>

  Run ROSA CLI tests against a different branch/repo:

  E2E_SKIP_ROSA_CLI=false \
  ROSA_REPO_URL="https://github.com/myuser/rosa" \
  ROSA_REPO_BRANCH="my-feature" \
  make ephemeral-e2e ID=<your-env-id>

  Run ROSA CLI tests with a different test profile:

  E2E_SKIP_ROSA_CLI=false \
  ROSA_TEST_PROFILE="rosa-hcp-extended" \
  make ephemeral-e2e ID=<your-env-id>

