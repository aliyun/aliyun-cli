#!/bin/bash
# Offline mock test for bin/aliyun-reload.
#
# No credentials, no network and no real ECS resources are used: a mock
# `aliyun` on PATH answers DescribeInstances from a local state file and
# records every mutating call, and `sleep` is stubbed out.
#
# Usage: bin/test/aliyun-reload-test.sh

set -u

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELOAD="$BIN_DIR/aliyun-reload"
COMMON="$BIN_DIR/aliyun-common"
REGION_IDS="$BIN_DIR/region_ids"
INSTANCE_ID="i-mock1234567890"
REGION_ID="cn-hangzhou"
IMAGE_ID="centos_7_9_x64_20G_alibase_mock.vhd"

CASE_NAME=""
CASE_DIR=""
CASE_DIRS=""
WORK=""
MOCKBIN=""
MOCK_IMAGE="$IMAGE_ID"
FORCE_ENV="FORCE=false"
EXTRA_ARGS=""
TIMEOUT_SECONDS=30
TIMED_OUT=0
TOTAL=0
FAILED=0

cleanup() {
    old_ifs=$IFS
    IFS='
'
    for dir in $CASE_DIRS; do
        [ -n "$dir" ] && rm -rf "$dir"
    done
    IFS=$old_ifs
}
trap cleanup EXIT

assert_true() {
    desc="$1"
    shift
    TOTAL=$((TOTAL + 1))
    if "$@" >/dev/null 2>&1; then
        echo "PASS  [$CASE_NAME] $desc"
    else
        echo "FAIL  [$CASE_NAME] $desc"
        FAILED=$((FAILED + 1))
    fi
}

assert_false() {
    desc="$1"
    shift
    TOTAL=$((TOTAL + 1))
    if "$@" >/dev/null 2>&1; then
        echo "FAIL  [$CASE_NAME] $desc"
        FAILED=$((FAILED + 1))
    else
        echo "PASS  [$CASE_NAME] $desc"
    fi
}

out_has() { grep -qF -- "$1" "$CASE_DIR/stdout"; }
out_lacks() { ! grep -qF -- "$1" "$CASE_DIR/stdout"; }
calls_has() { grep -qF -- "$1" "$CASE_DIR/calls.log"; }
calls_lack() { ! grep -qF -- "$1" "$CASE_DIR/calls.log"; }

mutating() {
    grep -cE 'Ecs (StopInstance|ReplaceSystemDisk|StartInstance)' "$CASE_DIR/calls.log" || true
}

mutating_count_is() { [ "$(mutating)" = "$1" ]; }

warning_before_stopping() {
    warn=$(grep -n 'replaces the system disk' "$CASE_DIR/stdout" | head -1 | cut -d: -f1)
    stop=$(grep -n 'MOCK_CALL Ecs StopInstance' "$CASE_DIR/stdout" | head -1 | cut -d: -f1)
    [ -n "$warn" ] && [ -n "$stop" ] && [ "$warn" -lt "$stop" ]
}

stop_replace_start_in_order() {
    stop=$(grep -n 'Ecs StopInstance' "$CASE_DIR/calls.log" | head -1 | cut -d: -f1)
    repl=$(grep -n 'Ecs ReplaceSystemDisk' "$CASE_DIR/calls.log" | head -1 | cut -d: -f1)
    start=$(grep -n 'Ecs StartInstance' "$CASE_DIR/calls.log" | head -1 | cut -d: -f1)
    [ -n "$stop" ] && [ -n "$repl" ] && [ -n "$start" ] &&
        [ "$stop" -lt "$repl" ] && [ "$repl" -lt "$start" ]
}

exit_code_is() { [ "$(cat "$CASE_DIR/exit_code")" = "$1" ]; }

new_case() {
    CASE_NAME="$1"
    MOCK_IMAGE="$IMAGE_ID"
    FORCE_ENV="FORCE=false"
    EXTRA_ARGS=""
    CASE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/aliyun-reload-test.XXXXXX")" || exit 1
    CASE_DIRS="${CASE_DIRS}${CASE_DIR}
"
    WORK="$CASE_DIR/work"
    MOCKBIN="$CASE_DIR/mockbin"
    mkdir -p "$WORK" "$MOCKBIN"
    cp "$COMMON" "$WORK/aliyun-common"
    cp "$REGION_IDS" "$WORK/region_ids"
    : > "$CASE_DIR/calls.log"
    : > "$CASE_DIR/sleep.log"
    printf 'Running\n' > "$CASE_DIR/status"

    cat > "$MOCKBIN/aliyun" <<'MOCK_ALIYUN'
#!/bin/bash
state="${MOCK_STATE_DIR:?MOCK_STATE_DIR missing}"
printf '%s\n' "$*" >> "$state/calls.log"
case "$1 $2" in
"Ecs DescribeInstances")
    printf '{"Instances":{"Instance":[{"InstanceId":"%s","RegionId":"%s","ImageId":"%s","Status":"%s","HostName":"mockhost","ZoneId":"%s-b","PublicIpAddress":{"IpAddress":["1.2.3.4"]},"NetworkInterfaces":{"NetworkInterface":[{"PrimaryIpAddress":"10.0.0.1"}]}}]}}\n' \
        "$MOCK_INSTANCE_ID" "$MOCK_REGION_ID" "$MOCK_IMAGE_ID" "$(cat "$state/status")" "$MOCK_REGION_ID"
    ;;
"Ecs StopInstance")
    printf 'MOCK_CALL %s\n' "$*"
    printf 'Stopped\n' > "$state/status"
    ;;
"Ecs ReplaceSystemDisk")
    printf 'MOCK_CALL %s\n' "$*"
    ;;
"Ecs StartInstance")
    printf 'MOCK_CALL %s\n' "$*"
    printf 'Running\n' > "$state/status"
    ;;
esac
exit 0
MOCK_ALIYUN

    cat > "$MOCKBIN/sleep" <<'MOCK_SLEEP'
#!/bin/bash
printf '%s\n' "$*" >> "${MOCK_STATE_DIR:?}/sleep.log"
exit 0
MOCK_SLEEP

    chmod +x "$MOCKBIN/aliyun" "$MOCKBIN/sleep"
}

# run_reload <stdin-line|EOF>
# A watchdog fails the case if the script regresses into an endless wait_until
# loop instead of hanging CI.
run_reload() {
    input="$1"
    : > "$CASE_DIR/calls.log"
    printf 'Running\n' > "$CASE_DIR/status"
    if [ "$input" = "EOF" ]; then
        : > "$CASE_DIR/stdin"
    else
        printf '%s\n' "$input" > "$CASE_DIR/stdin"
    fi

    TIMED_OUT=0
    (cd "$WORK" && exec env PATH="$MOCKBIN:$PATH" MOCK_STATE_DIR="$CASE_DIR" \
        MOCK_INSTANCE_ID="$INSTANCE_ID" MOCK_REGION_ID="$REGION_ID" MOCK_IMAGE_ID="$MOCK_IMAGE" \
        $FORCE_ENV bash "$RELOAD" "$INSTANCE_ID" $EXTRA_ARGS) \
        <"$CASE_DIR/stdin" >"$CASE_DIR/stdout" 2>"$CASE_DIR/stderr" &
    child=$!
    (
        waited=0
        while [ "$waited" -lt "$TIMEOUT_SECONDS" ]; do
            sleep 1
            kill -0 "$child" 2>/dev/null || exit 0
            waited=$((waited + 1))
        done
        kill -TERM "$child" 2>/dev/null
        sleep 1
        kill -KILL "$child" 2>/dev/null
    ) &
    watchdog=$!
    wait "$child"
    rc=$?
    kill "$watchdog" 2>/dev/null
    wait "$watchdog" 2>/dev/null
    if [ "$rc" -eq 143 ] || [ "$rc" -eq 137 ]; then
        TIMED_OUT=1
        rc=124
    fi
    echo "$rc" > "$CASE_DIR/exit_code"
    if [ "$TIMED_OUT" -eq 1 ]; then
        TOTAL=$((TOTAL + 1))
        FAILED=$((FAILED + 1))
        echo "FAIL  [$CASE_NAME] execution timed out after ${TIMEOUT_SECONDS}s"
    fi
}

echo "== bin/aliyun-reload offline mock test =="

# ---------------------------------------------------------------- confirm: y
new_case "confirm-y"
run_reload "y"
assert_true  "prompt states the system disk is replaced"      out_has "replaces the system disk"
assert_true  "prompt says it is not a plain restart"          out_has "NOT a plain restart"
assert_true  "prompt warns about data loss and backup"        out_has "back up the instance"
assert_true  "prompt shows the resolved instance id"          out_has "Instance: $INSTANCE_ID"
assert_true  "prompt shows the resolved region"               out_has "Region: $REGION_ID"
assert_true  "prompt shows the target image"                  out_has "Target image: $IMAGE_ID"
assert_false "prompt does not claim a plain restart"          out_has "Confirm to restart"
assert_true  "warning is printed before any resource change"  warning_before_stopping
assert_true  "stop, replace and start are called in order"    stop_replace_start_in_order
assert_true  "replace uses the displayed instance and image"  calls_has "ReplaceSystemDisk --InstanceId $INSTANCE_ID --ImageId $IMAGE_ID"
assert_true  "stop targets the displayed instance and region" calls_has "StopInstance --RegionId $REGION_ID --InstanceId $INSTANCE_ID"
assert_true  "start targets the displayed instance and region" calls_has "StartInstance --RegionId $REGION_ID --InstanceId $INSTANCE_ID"
assert_true  "three mutating calls were made"                 mutating_count_is 3
assert_true  "script exits 0"                                 exit_code_is 0

# -------------------------------------------------------------- confirm: yes
new_case "confirm-yes"
run_reload "yes"
assert_true  "yes also proceeds"                              mutating_count_is 3
assert_true  "stop, replace and start are called in order"    stop_replace_start_in_order
assert_true  "script exits 0"                                 exit_code_is 0

# ---------------------------------------------------------------- FORCE=true
new_case "force-true"
FORCE_ENV="FORCE=true"
run_reload "EOF"
assert_true  "FORCE still prints the replacement warning"     out_has "WARNING"
assert_true  "FORCE warning mentions data loss"               out_has "back up the instance"
assert_false "FORCE does not prompt for confirmation"         out_has "Confirm to replace the system disk"
assert_true  "FORCE warns before any resource change"         warning_before_stopping
assert_true  "FORCE stop targets the displayed instance"      calls_has "StopInstance --RegionId $REGION_ID --InstanceId $INSTANCE_ID"
assert_true  "FORCE proceeds without a terminal"              mutating_count_is 3
assert_true  "script exits 0"                                 exit_code_is 0

# ------------------------------------------------------------------ reject: n
new_case "reject-n"
run_reload "n"
assert_true  "n makes no mutating calls"                      mutating_count_is 0
assert_false "n never replaces the system disk"               calls_has "ReplaceSystemDisk"
assert_true  "warning is still shown before declining"        out_has "replaces the system disk"
assert_true  "script exits 0"                                 exit_code_is 0

# -------------------------------------------------------- reject: other input
new_case "reject-other"
run_reload "maybe"
assert_true  "non-confirmation input makes no mutating calls" mutating_count_is 0
assert_false "non-confirmation input never replaces the disk" calls_has "ReplaceSystemDisk"
assert_true  "script exits 0"                                 exit_code_is 0

# ------------------------------------------------------------------- reject: EOF
new_case "eof"
run_reload "EOF"
assert_true  "EOF makes no mutating calls"                    mutating_count_is 0
assert_false "EOF never replaces the system disk"             calls_has "ReplaceSystemDisk"
assert_true  "EOF is non-blocking and exits 0"                exit_code_is 0

# --------------------------------------------------- image not resolved (A2)
# Only the warning text is asserted here; the unresolved-image confirm path is
# pre-existing behaviour and is not claimed safe by this test.
new_case "image-not-resolved"
MOCK_IMAGE=""
run_reload "n"
assert_true  "unresolved image is reported honestly"          out_has "Target image: not resolved"
assert_false "unresolved image is not fabricated"             out_has "Target image: $IMAGE_ID"

echo "== result: $((TOTAL - FAILED))/$TOTAL assertions passed =="
if [ "$FAILED" -ne 0 ]; then
    echo "FAILED: $FAILED assertion(s)"
    exit 1
fi
echo "OK"
