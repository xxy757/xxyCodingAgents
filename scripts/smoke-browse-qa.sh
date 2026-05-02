#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG_FILE="${CONFIG_FILE:-${ROOT_DIR}/configs/config.yaml}"
HTTP_ADDR="${HTTP_ADDR:-:18080}"
BASE_URL="${BASE_URL:-http://127.0.0.1:18080}"
PPROF_ADDR="${PPROF_ADDR:-127.0.0.1:16060}"

for cmd in go curl jq sqlite3; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "[smoke] missing required command: ${cmd}" >&2
    exit 1
  fi
done

mkdir -p "${ROOT_DIR}/tmp"
TMP_DIR="$(mktemp -d "${ROOT_DIR}/tmp/smoke-browse-qa.XXXXXX")"
DATA_DIR="${TMP_DIR}/data"
mkdir -p "${DATA_DIR}"

DB_PATH="${DATA_DIR}/app.db"
WORKSPACE_ROOT="${DATA_DIR}/workspaces"
LOG_ROOT="${DATA_DIR}/logs"
CHECKPOINT_ROOT="${DATA_DIR}/checkpoints"
AGENT_RUNTIME_BASE_DIR="${DATA_DIR}/agent-runtime"
SERVER_LOG="${TMP_DIR}/server.log"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

resolve_browse_cli_path() {
  local raw path
  raw="$(awk -F: '/^[[:space:]]*browse_cli_path[[:space:]]*:/ {sub(/^[[:space:]]*/, "", $2); gsub(/"/, "", $2); print $2; exit}' "${CONFIG_FILE}" || true)"
  path="${BROWSE_CLI_PATH:-${raw}}"
  if [[ -z "${path}" ]]; then
    echo ""
    return
  fi
  if [[ "${path}" != /* ]]; then
    path="$(cd "${ROOT_DIR}" && cd "$(dirname "${path}")" && pwd)/$(basename "${path}")"
  fi
  echo "${path}"
}

BROWSE_CLI_PATH="$(resolve_browse_cli_path)"
if [[ -z "${BROWSE_CLI_PATH}" ]]; then
  echo "[smoke] browse_cli_path is empty (config/env), abort" >&2
  exit 1
fi
if [[ ! -x "${BROWSE_CLI_PATH}" ]]; then
  echo "[smoke] browse cli is not executable: ${BROWSE_CLI_PATH}" >&2
  exit 1
fi

echo "[smoke] using browse cli: ${BROWSE_CLI_PATH}"

run_direct_cli_smoke() {
  local workspace_path state_file screenshot_file
  workspace_path="${WORKSPACE_ROOT}/direct-cli-smoke"
  mkdir -p "${workspace_path}"

  echo "[smoke] tmux not found, fallback to direct browse CLI smoke"

  export BROWSE_STATE_FILE="${workspace_path}/.gstack/browse.json"
  (
    cd "${workspace_path}"
    "${BROWSE_CLI_PATH}" newtab https://example.com
    "${BROWSE_CLI_PATH}" screenshot qa-smoke.png
  )

  state_file="${workspace_path}/.gstack/browse.json"
  screenshot_file="${workspace_path}/qa-smoke.png"

  if [[ ! -f "${state_file}" ]]; then
    echo "[smoke] browse state file missing: ${state_file}" >&2
    exit 1
  fi
  if [[ ! -f "${screenshot_file}" ]]; then
    echo "[smoke] screenshot missing: ${screenshot_file}" >&2
    exit 1
  fi

  echo "[smoke] PASS (direct mode)"
  echo "[smoke] workspace=${workspace_path}"
  echo "[smoke] state_file=${state_file}"
  echo "[smoke] screenshot=${screenshot_file}"
}

run_scheduler_smoke() {
  local project_payload project_json project_id run_payload run_json run_id
  local workspace_path task_id task_title task_description task_command
  local task_status tasks_json state_file screenshot_file

  echo "[smoke] starting server with isolated data dir: ${TMP_DIR}"
  (
    cd "${ROOT_DIR}"
    AI_DEV_HTTP_ADDR="${HTTP_ADDR}" \
    AI_DEV_PPROF_ADDR="${PPROF_ADDR}" \
    AI_DEV_SQLITE_PATH="${DB_PATH}" \
    AI_DEV_WORKSPACE_ROOT="${WORKSPACE_ROOT}" \
    AI_DEV_LOG_ROOT="${LOG_ROOT}" \
    AI_DEV_CHECKPOINT_ROOT="${CHECKPOINT_ROOT}" \
    AI_DEV_AGENT_RUNTIME_BASE_DIR="${AGENT_RUNTIME_BASE_DIR}" \
    AI_DEV_BROWSE_CLI_PATH="${BROWSE_CLI_PATH}" \
    go run ./cmd/server -config "${CONFIG_FILE}" >"${SERVER_LOG}" 2>&1
  ) &
  SERVER_PID=$!

  echo "[smoke] waiting for /healthz"
  for _ in $(seq 1 90); do
    if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if ! curl -fsS "${BASE_URL}/readyz" >/dev/null 2>&1; then
    echo "[smoke] server not ready, tailing logs:" >&2
    tail -n 120 "${SERVER_LOG}" >&2 || true
    exit 1
  fi

  project_payload='{"name":"smoke-browse-qa","repo_url":"","description":"browse qa smoke"}'
  project_json="$(curl -fsS -X POST "${BASE_URL}/api/projects" -H 'Content-Type: application/json' -d "${project_payload}")"
  project_id="$(echo "${project_json}" | jq -r '.id')"
  if [[ -z "${project_id}" || "${project_id}" == "null" ]]; then
    echo "[smoke] failed to create project: ${project_json}" >&2
    exit 1
  fi

  run_payload="$(jq -nc --arg pid "${project_id}" '{"project_id":$pid,"title":"browse-qa-smoke","description":"browse smoke run"}')"
  run_json="$(curl -fsS -X POST "${BASE_URL}/api/runs" -H 'Content-Type: application/json' -d "${run_payload}")"
  run_id="$(echo "${run_json}" | jq -r '.id')"
  if [[ -z "${run_id}" || "${run_id}" == "null" ]]; then
    echo "[smoke] failed to create run: ${run_json}" >&2
    exit 1
  fi

  workspace_path="${WORKSPACE_ROOT}/${run_id}"
  mkdir -p "${workspace_path}"

  task_id="$(uuidgen | tr '[:upper:]' '[:lower:]')"
  task_title="browser qa smoke task"
  task_description="verify browse daemon startup and screenshot"
  task_command="browse newtab https://example.com && browse screenshot qa-smoke.png"

  sql_escape() {
    printf "%s" "$1" | sed "s/'/''/g"
  }

  sqlite3 "${DB_PATH}" <<SQL
INSERT INTO tasks (
  id, run_id, task_spec_id, task_type, attempt_no, status, priority, queue_status,
  resource_class, preemptible, restart_policy, title, description, input_data, output_data,
  workspace_path, parent_task_id, started_at, completed_at, created_at, updated_at
) VALUES (
  '$(sql_escape "${task_id}")',
  '$(sql_escape "${run_id}")',
  '',
  'qa',
  1,
  'queued',
  'normal',
  'queued',
  'light',
  1,
  'never',
  '$(sql_escape "${task_title}")',
  '$(sql_escape "${task_description}")',
  '$(sql_escape "${task_command}")',
  '',
  '$(sql_escape "${workspace_path}")',
  NULL,
  NULL,
  NULL,
  datetime('now'),
  datetime('now')
);
SQL

  echo "[smoke] task queued: ${task_id}"

  task_status=""
  for _ in $(seq 1 180); do
    tasks_json="$(curl -fsS "${BASE_URL}/api/runs/${run_id}/tasks")"
    task_status="$(echo "${tasks_json}" | jq -r --arg tid "${task_id}" '.[] | select(.id == $tid) | .status')"
    if [[ "${task_status}" == "completed" ]]; then
      break
    fi
    if [[ "${task_status}" == "failed" || "${task_status}" == "cancelled" || "${task_status}" == "evicted" ]]; then
      echo "[smoke] task ended unexpectedly: ${task_status}" >&2
      tail -n 160 "${SERVER_LOG}" >&2 || true
      exit 1
    fi
    sleep 2
  done

  if [[ "${task_status}" != "completed" ]]; then
    echo "[smoke] task did not complete in time, last status: ${task_status:-<empty>}" >&2
    tail -n 160 "${SERVER_LOG}" >&2 || true
    exit 1
  fi

  state_file="${workspace_path}/.gstack/browse.json"
  screenshot_file="${workspace_path}/qa-smoke.png"

  if [[ ! -f "${state_file}" ]]; then
    echo "[smoke] browse state file missing: ${state_file}" >&2
    exit 1
  fi
  if [[ ! -f "${screenshot_file}" ]]; then
    echo "[smoke] screenshot missing: ${screenshot_file}" >&2
    exit 1
  fi

  echo "[smoke] PASS (scheduler mode)"
  echo "[smoke] run_id=${run_id}"
  echo "[smoke] workspace=${workspace_path}"
  echo "[smoke] state_file=${state_file}"
  echo "[smoke] screenshot=${screenshot_file}"
}

if ! command -v tmux >/dev/null 2>&1; then
  run_direct_cli_smoke
  exit 0
fi

run_scheduler_smoke
