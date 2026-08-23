#!/usr/bin/env bash

set -uo pipefail

NAMESPACE="managed-workloads"
RESULTS_FILE="failure-results.csv"
TIMEOUT_SECONDS=180

# Write the CSV header. This overwrites results from an older run.
printf '%s\n' \
  "test,failure_type,workload,duration_seconds,result" \
  > "$RESULTS_FILE"

get_recovery_count() {
  local workload_name="$1"
  local count

  count="$(kubectl get managedworkload "$workload_name" \
    --namespace "$NAMESPACE" \
    --output jsonpath='{.status.recoveryCount}' \
    2>/dev/null || true)"

  if [[ ! "$count" =~ ^[0-9]+$ ]]; then
    count=0
  fi

  printf '%s' "$count"
}

get_phase() {
  local workload_name="$1"

  kubectl get managedworkload "$workload_name" \
    --namespace "$NAMESPACE" \
    --output jsonpath='{.status.phase}' \
    2>/dev/null || true
}

get_ready_replicas() {
  local workload_name="$1"
  local ready

  ready="$(kubectl get managedworkload "$workload_name" \
    --namespace "$NAMESPACE" \
    --output jsonpath='{.status.readyReplicas}' \
    2>/dev/null || true)"

  if [[ ! "$ready" =~ ^[0-9]+$ ]]; then
    ready=0
  fi

  printf '%s' "$ready"
}

get_desired_replicas() {
  local workload_name="$1"
  local desired

  desired="$(kubectl get managedworkload "$workload_name" \
    --namespace "$NAMESPACE" \
    --output jsonpath='{.spec.replicas}' \
    2>/dev/null || true)"

  if [[ ! "$desired" =~ ^[0-9]+$ ]]; then
    desired=0
  fi

  printf '%s' "$desired"
}

wait_until_initially_healthy() {
  local workload_name="$1"
  local deadline
  local phase
  local ready
  local desired

  deadline=$((SECONDS + TIMEOUT_SECONDS))

  while ((SECONDS < deadline)); do
    phase="$(get_phase "$workload_name")"
    ready="$(get_ready_replicas "$workload_name")"
    desired="$(get_desired_replicas "$workload_name")"

    if [[ "$phase" == "Healthy" ]] &&
      ((ready == desired)) &&
      ((desired > 0)); then
      return 0
    fi

    sleep 2
  done

  printf 'Timed out waiting for %s to become initially healthy\n' \
    "$workload_name" >&2

  return 1
}

wait_for_recovery() {
  local workload_name="$1"
  local previous_count="$2"
  local deadline
  local phase
  local ready
  local desired
  local current_count

  deadline=$((SECONDS + TIMEOUT_SECONDS))

  while ((SECONDS < deadline)); do
    phase="$(get_phase "$workload_name")"
    ready="$(get_ready_replicas "$workload_name")"
    desired="$(get_desired_replicas "$workload_name")"
    current_count="$(get_recovery_count "$workload_name")"

    printf \
      'Waiting: workload=%s phase=%s ready=%s/%s recovery=%s previous=%s\n' \
      "$workload_name" \
      "$phase" \
      "$ready" \
      "$desired" \
      "$current_count" \
      "$previous_count"

    if [[ "$phase" == "Healthy" ]] &&
      ((ready == desired)) &&
      ((desired > 0)) &&
      ((current_count > previous_count)); then
      return 0
    fi

    sleep 2
  done

  printf 'Timed out waiting for recovery of %s\n' \
    "$workload_name" >&2

  return 1
}

record_result() {
  local test_number="$1"
  local failure_type="$2"
  local workload_name="$3"
  local duration="$4"
  local result="$5"

  printf '%s,%s,%s,%s,%s\n' \
    "$test_number" \
    "$failure_type" \
    "$workload_name" \
    "$duration" \
    "$result" \
    >> "$RESULTS_FILE"

  printf '\nTest %s: %s on %s - %s (%ss)\n\n' \
    "$test_number" \
    "$failure_type" \
    "$workload_name" \
    "$result" \
    "$duration"
}

run_pod_delete_test() {
  local test_number="$1"
  local workload_name="$2"
  local pod_name
  local previous_count
  local started_at
  local duration
  local result

  printf '\nStarting test %s: delete Pod for %s\n' \
    "$test_number" \
    "$workload_name"

  if ! wait_until_initially_healthy "$workload_name"; then
    record_result \
      "$test_number" \
      "pod-delete" \
      "$workload_name" \
      "0" \
      "FAIL"

    return
  fi

  previous_count="$(get_recovery_count "$workload_name")"

  pod_name="$(kubectl get pods \
    --namespace "$NAMESPACE" \
    --selector "app.kubernetes.io/name=$workload_name" \
    --output jsonpath='{.items[0].metadata.name}' \
    2>/dev/null || true)"

  if [[ -z "$pod_name" ]]; then
    printf 'Could not find a Pod for %s\n' \
      "$workload_name" >&2

    record_result \
      "$test_number" \
      "pod-delete" \
      "$workload_name" \
      "0" \
      "FAIL"

    return
  fi

  started_at="$(date +%s)"

  if ! kubectl delete pod "$pod_name" \
    --namespace "$NAMESPACE" \
    --wait=false; then
    record_result \
      "$test_number" \
      "pod-delete" \
      "$workload_name" \
      "0" \
      "FAIL"

    return
  fi

  if wait_for_recovery "$workload_name" "$previous_count"; then
    result="PASS"
  else
    result="FAIL"
  fi

  duration=$(($(date +%s) - started_at))

  record_result \
    "$test_number" \
    "pod-delete" \
    "$workload_name" \
    "$duration" \
    "$result"
}

run_deployment_delete_test() {
  local test_number="$1"
  local workload_name="$2"
  local previous_count
  local started_at
  local duration
  local result

  printf '\nStarting test %s: delete Deployment for %s\n' \
    "$test_number" \
    "$workload_name"

  if ! wait_until_initially_healthy "$workload_name"; then
    record_result \
      "$test_number" \
      "deployment-delete" \
      "$workload_name" \
      "0" \
      "FAIL"

    return
  fi

  previous_count="$(get_recovery_count "$workload_name")"
  started_at="$(date +%s)"

  if ! kubectl delete deployment "$workload_name" \
    --namespace "$NAMESPACE" \
    --wait=true; then
    record_result \
      "$test_number" \
      "deployment-delete" \
      "$workload_name" \
      "0" \
      "FAIL"

    return
  fi

  if wait_for_recovery "$workload_name" "$previous_count"; then
    result="PASS"
  else
    result="FAIL"
  fi

  duration=$(($(date +%s) - started_at))

  record_result \
    "$test_number" \
    "deployment-delete" \
    "$workload_name" \
    "$duration" \
    "$result"
}

run_replica_drift_test() {
  local test_number="$1"
  local workload_name="$2"
  local previous_count
  local started_at
  local duration
  local result

  printf '\nStarting test %s: replica drift for %s\n' \
    "$test_number" \
    "$workload_name"

  if ! wait_until_initially_healthy "$workload_name"; then
    record_result \
      "$test_number" \
      "replica-drift" \
      "$workload_name" \
      "0" \
      "FAIL"

    return
  fi

  previous_count="$(get_recovery_count "$workload_name")"
  started_at="$(date +%s)"

  if ! kubectl scale deployment "$workload_name" \
    --namespace "$NAMESPACE" \
    --replicas=0; then
    record_result \
      "$test_number" \
      "replica-drift" \
      "$workload_name" \
      "0" \
      "FAIL"

    return
  fi

  if wait_for_recovery "$workload_name" "$previous_count"; then
    result="PASS"
  else
    result="FAIL"
  fi

  duration=$(($(date +%s) - started_at))

  record_result \
    "$test_number" \
    "replica-drift" \
    "$workload_name" \
    "$duration" \
    "$result"
}

printf 'Checking whether all 20 workloads exist...\n'

workload_count="$(kubectl get managedworkloads \
  --namespace "$NAMESPACE" \
  --no-headers \
  2>/dev/null |
  wc -l |
  tr -d ' ')"

if [[ "$workload_count" != "20" ]]; then
  printf 'Expected 20 ManagedWorkloads, but found %s\n' \
    "$workload_count" >&2
  exit 1
fi

printf 'Found 20 ManagedWorkloads.\n'

test_number=0

# Tests 1-10: delete one Pod from workloads 01-10.
for index in $(seq 1 10); do
  test_number=$((test_number + 1))
  workload_name="$(printf 'workload-%02d' "$index")"

  run_pod_delete_test \
    "$test_number" \
    "$workload_name"
done

# Tests 11-20: delete Deployments for workloads 11-20.
for index in $(seq 11 20); do
  test_number=$((test_number + 1))
  workload_name="$(printf 'workload-%02d' "$index")"

  run_deployment_delete_test \
    "$test_number" \
    "$workload_name"
done

# Tests 21-30: introduce replica drift for workloads 01-10.
for index in $(seq 1 10); do
  test_number=$((test_number + 1))
  workload_name="$(printf 'workload-%02d' "$index")"

  run_replica_drift_test \
    "$test_number" \
    "$workload_name"
done

passed_count="$(awk -F, '
  NR > 1 && $5 == "PASS" {
    passed++
  }
  END {
    print passed + 0
  }
' "$RESULTS_FILE")"

failed_count="$(awk -F, '
  NR > 1 && $5 == "FAIL" {
    failed++
  }
  END {
    print failed + 0
  }
' "$RESULTS_FILE")"

printf '\nFailure testing complete.\n'
printf 'Passed: %s\n' "$passed_count"
printf 'Failed: %s\n' "$failed_count"
printf 'Results: %s\n' "$RESULTS_FILE"

if ((failed_count > 0)); then
  exit 1
fi