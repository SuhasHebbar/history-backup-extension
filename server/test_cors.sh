#!/usr/bin/env bash
set -euo pipefail

ADDR="127.0.0.1:9099"
BASE="http://$ADDR"
TEST_ORIGIN="http://test.example.com"
PASS=0
FAIL=0

cleanup() {
    if [[ -n "${SERVER_PID:-}" ]]; then
        kill "$SERVER_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Start server
go run . --working-directory ./ --addr "$ADDR" --allowed-origins "$TEST_ORIGIN" \
    > /tmp/cors-test-server.log 2>&1 &
SERVER_PID=$!

# Wait for it to be ready
for i in $(seq 1 20); do
    if curl -s "$BASE/status" > /dev/null 2>&1; then break; fi
    sleep 0.2
done

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1 — $2"; FAIL=$((FAIL + 1)); }

check_header_present() {
    local label="$1" response="$2" header="$3" expected="$4"
    if echo "$response" | grep -qi "^$header: $expected"; then
        pass "$label"
    else
        fail "$label" "expected '$header: $expected' not found in response headers"
        echo "$response" | grep -i "^$header" || true
    fi
}

check_header_absent() {
    local label="$1" response="$2" header="$3"
    if echo "$response" | grep -qi "^$header:"; then
        fail "$label" "unexpected '$header' header found"
        echo "$response" | grep -i "^$header"
    else
        pass "$label"
    fi
}

check_status() {
    local label="$1" response="$2" expected_code="$3"
    local actual
    actual=$(echo "$response" | head -1 | awk '{print $2}')
    if [[ "$actual" == "$expected_code" ]]; then
        pass "$label"
    else
        fail "$label" "expected HTTP $expected_code, got $actual"
    fi
}

# ── Test 1: No Origin → no ACAO ──────────────────────────────────────────────
echo "Test 1: No Origin header"
r=$(curl -si "$BASE/status")
check_status       "HTTP 200"              "$r" 200
check_header_absent "No ACAO header"       "$r" "Access-Control-Allow-Origin"

# ── Test 2: Matching origin → ACAO present ────────────────────────────────────
echo "Test 2: Matching Origin"
r=$(curl -si -H "Origin: $TEST_ORIGIN" "$BASE/status")
check_status        "HTTP 200"              "$r" 200
check_header_present "ACAO matches origin"  "$r" "Access-Control-Allow-Origin" "$TEST_ORIGIN"

# ── Test 3: Non-matching origin → no ACAO ────────────────────────────────────
echo "Test 3: Non-matching Origin"
r=$(curl -si -H "Origin: http://evil.example.com" "$BASE/status")
check_status        "HTTP 200"              "$r" 200
check_header_absent "No ACAO header"        "$r" "Access-Control-Allow-Origin"

# ── Test 4: Preflight OPTIONS + GET (allowed) ─────────────────────────────────
echo "Test 4: Preflight OPTIONS + GET (allowed method)"
r=$(curl -si -X OPTIONS \
    -H "Origin: $TEST_ORIGIN" \
    -H "Access-Control-Request-Method: GET" \
    "$BASE/status")
check_header_present "ACAO present"         "$r" "Access-Control-Allow-Origin" "$TEST_ORIGIN"
check_header_present "ACAM contains GET"    "$r" "Access-Control-Allow-Methods" "GET"

# ── Test 5: Preflight OPTIONS + PUT (disallowed) ──────────────────────────────
echo "Test 5: Preflight OPTIONS + PUT (disallowed method)"
r=$(curl -si -X OPTIONS \
    -H "Origin: $TEST_ORIGIN" \
    -H "Access-Control-Request-Method: PUT" \
    "$BASE/status")
check_header_absent "No ACAO header"        "$r" "Access-Control-Allow-Origin"

# ── Test 6: POST /status → 405 ───────────────────────────────────────────────
echo "Test 6: POST to /status (method not allowed)"
r=$(curl -si -X POST -H "Origin: $TEST_ORIGIN" "$BASE/status")
check_status        "HTTP 405"              "$r" 405

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "Results: $PASS passed, $FAIL failed"
[[ $FAIL -eq 0 ]]
