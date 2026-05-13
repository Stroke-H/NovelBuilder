#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
API_DIR="$ROOT_DIR/apps/api"
WEB_DIR="$ROOT_DIR/apps/web"
LOG_DIR="$ROOT_DIR/logs"
RUN_DIR="$ROOT_DIR/.tmp"

API_PORT="${NOVEL_GENERATER_BACKEND_PORT:-19081}"
WEB_PORT="${NOVEL_GENERATER_WEB_PORT:-5174}"
API_HOST="${NOVEL_GENERATER_BACKEND_HOST:-0.0.0.0}"
WEB_HOST="${NOVEL_GENERATER_WEB_HOST:-0.0.0.0}"
AUTO_INSTALL_DEPS="${NOVEL_GENERATER_AUTO_INSTALL_DEPS:-0}"
KILL_PREVIOUS="${NOVEL_GENERATER_KILL_PREVIOUS:-0}"

if [[ -t 1 ]]; then
  GREEN=$'\033[0;32m'
  RED=$'\033[0;31m'
  YELLOW=$'\033[1;33m'
  BLUE=$'\033[0;34m'
  NC=$'\033[0m'
else
  GREEN=""
  RED=""
  YELLOW=""
  BLUE=""
  NC=""
fi

mkdir -p "$LOG_DIR"
mkdir -p "$RUN_DIR"

STARTED_AT="$(date '+%Y%m%d-%H%M%S')"
MAIN_LOG="$LOG_DIR/start-$STARTED_AT.log"
API_LOG="$LOG_DIR/api-$STARTED_AT.log"
WEB_LOG="$LOG_DIR/web-$STARTED_AT.log"
CURRENT_ENV_FILE="$RUN_DIR/current-dev.env"

API_PID=""
WEB_PID=""

log() {
  local level="$1"
  shift
  printf '[%s] [%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$level" "$*" | tee -a "$MAIN_LOG" >&2
}

print_line() {
  printf '%b\n' "$*" | tee -a "$MAIN_LOG"
}

fail() {
  log "ERROR" "$*"
  log "ERROR" "启动已中止。主日志: $MAIN_LOG"
  exit 1
}

require_command() {
  local command_name="$1"
  command -v "$command_name" >/dev/null 2>&1 || fail "缺少依赖命令: $command_name"
  log "INFO" "依赖命令可用: $command_name"
}

check_dir() {
  local dir="$1"
  [[ -d "$dir" ]] || fail "目录不存在: $dir"
}

check_file() {
  local file="$1"
  [[ -f "$file" ]] || fail "文件不存在: $file"
}

http_status() {
  local url="$1"
  curl --noproxy "*" -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" || true
}

tail_log() {
  local file="$1"
  if [[ -f "$file" ]]; then
    log "INFO" "最近日志片段: $file"
    tail -n 80 "$file" >>"$MAIN_LOG" 2>&1 || true
  fi
}

get_lan_ips() {
  {
    local default_iface
    default_iface=$(route get default 2>/dev/null | awk '/interface:/{print $2; exit}')

    if [[ -n "$default_iface" ]]; then
      local default_ip
      default_ip=$(ipconfig getifaddr "$default_iface" 2>/dev/null || true)
      if [[ -n "$default_ip" ]]; then
        echo "$default_ip"
      fi
    fi

    ifconfig 2>/dev/null | awk '/inet / && $2 !~ /^127\./ && $2 !~ /^198\.18\./ {print $2}'
  } | awk '!seen[$0]++'
}

get_mdns_host() {
  local local_host_name
  local_host_name=$(scutil --get LocalHostName 2>/dev/null || true)

  if [[ -n "${NOVEL_GENERATER_LAN_HOST:-}" ]]; then
    echo "$NOVEL_GENERATER_LAN_HOST"
    return 0
  fi

  if [[ -n "$local_host_name" ]]; then
    printf "%s.local\n" "$local_host_name" | tr '[:upper:]' '[:lower:]'
    return 0
  fi

  echo "localhost"
}

port_pid() {
  local port="$1"
  command -v lsof >/dev/null 2>&1 || return 1
  lsof -t -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | head -n 1 || true
}

is_port_free() {
  local port="$1"
  [[ -z "$(port_pid "$port")" ]]
}

find_available_port() {
  local start_port="$1"
  local max_offset="${2:-80}"
  local port="$start_port"
  local offset=0

  while [[ "$offset" -le "$max_offset" ]]; do
    if is_port_free "$port"; then
      echo "$port"
      return 0
    fi

    port=$((port + 1))
    offset=$((offset + 1))
  done

  return 1
}

choose_port() {
  local requested_port="$1"
  local fallback_start="$2"
  local expected_keyword="$3"
  local service_name="$4"
  local pid

  pid="$(port_pid "$requested_port")"
  if [[ -z "$pid" ]]; then
    log "INFO" "$service_name 端口可用: $requested_port"
    echo "$requested_port"
    return 0
  fi

  local command_full
  command_full=$(ps -p "$pid" -o command= 2>/dev/null | head -n 1 || true)
  log "WARN" "$service_name 端口 $requested_port 已被 PID $pid 占用: ${command_full:0:120}"

  if [[ "$KILL_PREVIOUS" = "1" ]] && echo "$command_full" | grep -qiE "$expected_keyword"; then
    log "WARN" "识别为旧的 $service_name 进程，按配置自动停止。"
    kill "$pid" >/dev/null 2>&1 || true
    sleep 1
    if is_port_free "$requested_port"; then
      log "INFO" "$service_name 端口已释放: $requested_port"
      echo "$requested_port"
      return 0
    fi
  fi

  local fallback_port
  fallback_port=$(find_available_port "$fallback_start") || fail "$service_name 无可用备用端口，起始端口: $fallback_start"
  log "WARN" "$service_name 将改用备用端口: $fallback_port"
  echo "$fallback_port"
}

check_npm_dependencies() {
  check_file "$WEB_DIR/package.json"
  check_file "$WEB_DIR/package-lock.json"

  if [[ ! -d "$WEB_DIR/node_modules" ]]; then
    if [[ "$AUTO_INSTALL_DEPS" = "1" ]]; then
      log "WARN" "前端 node_modules 不存在，开始执行 npm install。"
      (cd "$WEB_DIR" && npm install >>"$MAIN_LOG" 2>&1) || fail "npm install 失败，详情见日志: $MAIN_LOG"
    else
      fail "前端依赖未安装: $WEB_DIR/node_modules 不存在。可在 apps/web 执行 npm install，或设置 NOVEL_GENERATER_AUTO_INSTALL_DEPS=1。"
    fi
  fi

  (cd "$WEB_DIR" && npm ls --depth=0 >>"$MAIN_LOG" 2>&1) || fail "前端依赖检查失败，详情见日志: $MAIN_LOG"
  log "INFO" "前端依赖检查通过"
}

check_go_dependencies() {
  check_file "$API_DIR/go.mod"
  check_file "$API_DIR/go.sum"

  if ! (cd "$API_DIR" && go list ./... >>"$MAIN_LOG" 2>&1); then
    if [[ "$AUTO_INSTALL_DEPS" = "1" ]]; then
      log "WARN" "Go 依赖检查失败，开始执行 go mod download。"
      (cd "$API_DIR" && go mod download >>"$MAIN_LOG" 2>&1) || fail "go mod download 失败，详情见日志: $MAIN_LOG"
      (cd "$API_DIR" && go list ./... >>"$MAIN_LOG" 2>&1) || fail "后端 Go 依赖检查失败，详情见日志: $MAIN_LOG"
    else
      fail "后端 Go 依赖检查失败，详情见日志: ${MAIN_LOG}。可设置 NOVEL_GENERATER_AUTO_INSTALL_DEPS=1 尝试自动下载。"
    fi
  fi

  log "INFO" "后端 Go 依赖检查通过"
}

check_runtime_configs() {
  check_file "$API_DIR/data/database_config.json"
  check_file "$API_DIR/data/ai_config.json"
  log "INFO" "运行配置文件检查通过"
}

stop_children() {
  if [[ -n "$API_PID" ]] && kill -0 "$API_PID" >/dev/null 2>&1; then
    log "INFO" "停止后端进程: $API_PID"
    kill "$API_PID" >/dev/null 2>&1 || true
  fi

  if [[ -n "$WEB_PID" ]] && kill -0 "$WEB_PID" >/dev/null 2>&1; then
    log "INFO" "停止前端进程: $WEB_PID"
    kill "$WEB_PID" >/dev/null 2>&1 || true
  fi
}

cleanup() {
  local status=$?
  stop_children
  if [[ "$status" -eq 0 ]]; then
    log "INFO" "所有服务已停止"
  fi
  exit "$status"
}

wait_for_backend_ready() {
  local url="http://127.0.0.1:$API_PORT/healthz"
  local max_attempts=30
  local attempt=1

  log "INFO" "等待后端健康检查: $url"
  while [[ "$attempt" -le "$max_attempts" ]]; do
    local status
    status=$(http_status "$url")

    if [[ "$status" = "200" ]]; then
      log "INFO" "后端已就绪: HTTP $status"
      return 0
    fi

    if ! kill -0 "$API_PID" >/dev/null 2>&1; then
      tail_log "$API_LOG"
      fail "后端进程在就绪前退出"
    fi

    sleep 1
    attempt=$((attempt + 1))
  done

  tail_log "$API_LOG"
  fail "后端在 ${max_attempts}s 内未就绪"
}

wait_for_frontend_ready() {
  local url="http://127.0.0.1:$WEB_PORT/"
  local max_attempts=30
  local attempt=1

  log "INFO" "等待前端健康检查: $url"
  while [[ "$attempt" -le "$max_attempts" ]]; do
    local status
    status=$(http_status "$url")

    if [[ "$status" = "200" ]]; then
      log "INFO" "前端已就绪: HTTP $status"
      return 0
    fi

    if ! kill -0 "$WEB_PID" >/dev/null 2>&1; then
      tail_log "$WEB_LOG"
      fail "前端进程在就绪前退出"
    fi

    sleep 1
    attempt=$((attempt + 1))
  done

  tail_log "$WEB_LOG"
  fail "前端在 ${max_attempts}s 内未就绪"
}

print_entrypoints() {
  local lan_host
  local lan_ips
  lan_host=$(get_mdns_host)
  lan_ips=$(get_lan_ips)

  print_line ""
  print_line "${GREEN}==========================================${NC}"
  print_line "${GREEN}  NovelGenerater is running${NC}"
  print_line "${GREEN}  Local Frontend: http://localhost:$WEB_PORT/${NC}"
  print_line "${GREEN}  Local Backend:  http://localhost:$API_PORT${NC}"
  print_line "${GREEN}  LAN Frontend:   http://$lan_host:$WEB_PORT/${NC}"
  print_line "${GREEN}  LAN Backend:    http://$lan_host:$API_PORT${NC}"

  if [[ -n "$lan_ips" ]]; then
    print_line "${GREEN}  LAN IP Fallback:${NC}"
    while IFS= read -r lan_ip; do
      [[ -n "$lan_ip" ]] && print_line "${GREEN}    http://$lan_ip:$WEB_PORT/${NC}"
    done <<< "$lan_ips"
  fi

  print_line "${YELLOW}  Press Ctrl+C to stop all services.${NC}"
  print_line "${GREEN}==========================================${NC}"
  print_line ""
}

write_current_env() {
  {
    printf 'NOVEL_GENERATER_BACKEND_PORT=%s\n' "$API_PORT"
    printf 'NOVEL_GENERATER_WEB_PORT=%s\n' "$WEB_PORT"
    printf 'NOVEL_GENERATER_BACKEND_URL=http://localhost:%s\n' "$API_PORT"
    printf 'NOVEL_GENERATER_WEB_URL=http://localhost:%s/\n' "$WEB_PORT"
  } >"$CURRENT_ENV_FILE"
  log "INFO" "当前访问地址已写入: $CURRENT_ENV_FILE"
}

trap cleanup INT TERM EXIT

print_line "${BLUE}==========================================${NC}"
print_line "${BLUE}  Starting NovelGenerater Environment${NC}"
print_line "${BLUE}==========================================${NC}"
log "INFO" "主日志: $MAIN_LOG"
log "INFO" "后端端口: ${API_PORT}，前端端口: ${WEB_PORT}"

check_dir "$API_DIR"
check_dir "$WEB_DIR"
require_command node
require_command npm
require_command go
require_command curl
check_runtime_configs
check_go_dependencies
check_npm_dependencies
API_PORT="$(choose_port "$API_PORT" 19082 "novel-generater|go run|/go-build" "Go Backend")"
WEB_PORT="$(choose_port "$WEB_PORT" 15174 "novel-generater|node|vite|esbuild" "Vue Frontend")"
log "INFO" "最终端口: 后端 ${API_PORT}，前端 ${WEB_PORT}"

log "INFO" "启动后端: $API_HOST:$API_PORT"
(
  cd "$API_DIR"
  NOVEL_GENERATER_BACKEND_HOST="$API_HOST" NOVEL_GENERATER_BACKEND_PORT="$API_PORT" go run . >>"$API_LOG" 2>&1
) &
API_PID=$!
log "INFO" "后端进程 PID: ${API_PID}，日志: ${API_LOG}"
wait_for_backend_ready

log "INFO" "启动前端: $WEB_HOST:$WEB_PORT"
(
  cd "$WEB_DIR"
  NOVEL_GENERATER_BACKEND_PORT="$API_PORT" NOVEL_GENERATER_WEB_HOST="$WEB_HOST" NOVEL_GENERATER_WEB_PORT="$WEB_PORT" npm run dev >>"$WEB_LOG" 2>&1
) &
WEB_PID=$!
log "INFO" "前端进程 PID: ${WEB_PID}，日志: ${WEB_LOG}"
wait_for_frontend_ready

write_current_env
print_entrypoints

while true; do
  if ! kill -0 "$API_PID" >/dev/null 2>&1; then
    tail_log "$API_LOG"
    fail "后端进程已退出，请查看日志: $API_LOG"
  fi

  if ! kill -0 "$WEB_PID" >/dev/null 2>&1; then
    tail_log "$WEB_LOG"
    fail "前端进程已退出，请查看日志: $WEB_LOG"
  fi

  sleep 2
done
