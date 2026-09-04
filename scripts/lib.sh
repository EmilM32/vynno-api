# Shared helpers for scripts/ in this repo. Source from those scripts only.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PIDFILE="$ROOT/var/api.pid"
DEV_PIDFILE="$ROOT/var/dev.pid"
COMPOSE=(docker compose --project-directory "$ROOT")
PROD_DB=vynno
DEV_DB=vynno_dev
DEFAULT_PROD_ADDR=127.0.0.1:27182
DEFAULT_DEV_ADDR=127.0.0.1:8081
API_LOG="$ROOT/logs/api.log"
LOG_MAX_BYTES="${LOG_MAX_BYTES:-1048576}"
LOG_KEEP="${LOG_KEEP:-7}"

die() {
	echo "$*" >&2
	exit 1
}

rotate_log() {
	local file="$1"
	local max_bytes="${2:-$LOG_MAX_BYTES}"
	local keep="${3:-$LOG_KEEP}"
	[[ -f "$file" ]] || return 0
	local size
	size="$(wc -c <"$file" | tr -d '[:space:]')"
	if [[ "$size" -lt "$max_bytes" ]]; then
		return 0
	fi
	mv "$file" "${file}.$(date +%Y%m%d%H%M%S)"
	local extras
	extras="$(
		{ ls -1t "$file".* 2>/dev/null || true; } | tail -n +"$((keep + 1))"
	)"
	if [[ -n "$extras" ]]; then
		while IFS= read -r old; do
			[[ -n "$old" ]] || continue
			rm -f "$old"
		done <<<"$extras"
	fi
}

require_env_file() {
	if [[ ! -f "$ROOT/.env" ]]; then
		die "missing $ROOT/.env — copy .env.example"
	fi
}

load_env() {
	require_env_file
	set -a
	# shellcheck disable=SC1091
	. "$ROOT/.env"
	set +a
}

require_var() {
	local name="$1"
	if [[ -z "${!name:-}" ]]; then
		die "missing $name in .env — see .env.example"
	fi
}

require_build() {
	if [[ ! -f "$ROOT/bin/vynno-api" ]]; then
		die "missing $ROOT/bin/vynno-api — run scripts/build first"
	fi
}

# postgres://user:pass@host:port/dbname?sslmode=disable → dbname
postgres_db_name() {
	local url="${1-}"
	local noquery name
	noquery="${url%%\?*}"
	name="${noquery##*/}"
	if [[ -z "$name" || "$name" == *":"* || "$name" == *"@"* ]]; then
		die "cannot parse database name from URL"
	fi
	printf '%s\n' "$name"
}

require_prod_database() {
	local name
	name="$(postgres_db_name "${DATABASE_URL-}")"
	if [[ "$name" != "$PROD_DB" ]]; then
		die "DATABASE_URL must target database $PROD_DB (production); got $name"
	fi
}

require_dev_database_url() {
	require_var DEV_DATABASE_URL
	local name
	name="$(postgres_db_name "$DEV_DATABASE_URL")"
	if [[ "$name" != "$DEV_DB" ]]; then
		die "DEV_DATABASE_URL must target database $DEV_DB; got $name"
	fi
}

alive_pid() {
	local pid="${1-}"
	[[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

pidfile_pid() {
	local file="$1"
	if [[ ! -f "$file" ]]; then
		return 1
	fi
	local pid
	pid="$(cat "$file")"
	if alive_pid "$pid"; then
		echo "$pid"
		return 0
	fi
	rm -f "$file"
	return 1
}

kill_pid_wait() {
	local pid="${1-}"
	if ! alive_pid "$pid"; then
		return 0
	fi
	kill "$pid" 2>/dev/null || true
	local i
	for i in $(seq 1 20); do
		if ! alive_pid "$pid"; then
			return 0
		fi
		sleep 0.1
	done
	kill -9 "$pid" 2>/dev/null || true
}

kill_pid_tree() {
	local pid="${1-}" child
	if ! alive_pid "$pid"; then
		return 0
	fi
	for child in $(pgrep -P "$pid" 2>/dev/null || true); do
		kill_pid_tree "$child"
	done
	kill_pid_wait "$pid"
}

prod_addr() {
	printf '%s\n' "${ADDR:-$DEFAULT_PROD_ADDR}"
}

prod_listen_port() {
	local addr
	addr="$(prod_addr)"
	printf '%s\n' "${addr##*:}"
}

api_pid() {
	local pid pids
	if pid="$(pidfile_pid "$PIDFILE")"; then
		echo "$pid"
		return 0
	fi
	if pids="$(listen_pids_on_port "$(prod_listen_port)")"; then
		read -r pid <<< "$pids"
		echo "$pid"
		return 0
	fi
	return 1
}

stop_api() {
	local pid pids
	if pid="$(pidfile_pid "$PIDFILE")"; then
		kill_pid_wait "$pid"
	fi
	if pids="$(listen_pids_on_port "$(prod_listen_port)")"; then
		while IFS= read -r pid; do
			[[ -n "$pid" ]] || continue
			kill_pid_wait "$pid"
		done <<<"$pids"
	fi
	rm -f "$PIDFILE"
}

# Probe 127.0.0.1 (IPv4). ADDR is 127.0.0.1:port; localhost can prefer ::1.
wait_for_healthz() {
	local port="${1-}" pid="${2-}" i
	if [[ -z "$port" ]]; then
		die "wait_for_healthz: missing port"
	fi
	for i in $(seq 1 40); do
		if [[ -n "$pid" ]] && ! alive_pid "$pid"; then
			die "API process ${pid} exited before /healthz — see $ROOT/logs/api.log"
		fi
		if curl -sf --ipv4 -m 1 "http://127.0.0.1:${port}/healthz" >/dev/null 2>&1; then
			return 0
		fi
		sleep 0.25
	done
	die "API did not become ready on 127.0.0.1:${port}/healthz — see $ROOT/logs/api.log"
}

dev_addr() {
	printf '%s\n' "${DEV_ADDR:-$DEFAULT_DEV_ADDR}"
}

dev_listen_port() {
	local addr
	addr="$(dev_addr)"
	printf '%s\n' "${addr##*:}"
}

# PIDs listening on the playground TCP port. go run's child is the listener;
# a leftover after the parent dies has no pidfile.
listen_pids_on_port() {
	local port="$1" out
	if [[ -z "$port" ]] || ! command -v lsof >/dev/null 2>&1; then
		return 1
	fi
	out="$(lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null || true)"
	if [[ -z "$out" ]]; then
		return 1
	fi
	printf '%s\n' "$out"
	return 0
}

dev_api_pid() {
	local pid
	if pid="$(pidfile_pid "$DEV_PIDFILE")"; then
		echo "$pid"
		return 0
	fi
	local pids
	if pids="$(listen_pids_on_port "$(dev_listen_port)")"; then
		# First PID is enough for "already running" messages.
		read -r pid <<< "$pids"
		echo "$pid"
		return 0
	fi
	return 1
}

stop_dev_api() {
	local pid pids
	if pid="$(pidfile_pid "$DEV_PIDFILE")"; then
		kill_pid_tree "$pid"
	fi
	if pids="$(listen_pids_on_port "$(dev_listen_port)")"; then
		while IFS= read -r pid; do
			[[ -n "$pid" ]] || continue
			kill_pid_tree "$pid"
		done <<<"$pids"
	fi
	rm -f "$DEV_PIDFILE"
}

wait_for_postgres() {
	local i
	for i in $(seq 1 40); do
		if "${COMPOSE[@]}" exec -T postgres pg_isready -U vynno -d vynno >/dev/null 2>&1; then
			return 0
		fi
		sleep 0.25
	done
	die "postgres did not become ready"
}

ensure_dev_database() {
	local exists
	exists="$("${COMPOSE[@]}" exec -T postgres psql -U vynno -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '${DEV_DB}'" | tr -d '[:space:]')"
	if [[ "$exists" != "1" ]]; then
		"${COMPOSE[@]}" exec -T postgres psql -U vynno -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE ${DEV_DB} OWNER vynno"
	fi
}

ensure_postgres() {
	"${COMPOSE[@]}" up -d
	wait_for_postgres
	ensure_dev_database
}
