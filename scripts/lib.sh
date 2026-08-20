# Shared helpers for scripts/ in this repo. Source from those scripts only.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PIDFILE="$ROOT/var/api.pid"
DEV_PIDFILE="$ROOT/var/dev.pid"
COMPOSE=(docker compose --project-directory "$ROOT")
PROD_DB=vynno
DEV_DB=vynno_dev
DEFAULT_DEV_ADDR=127.0.0.1:8081

die() {
	echo "$*" >&2
	exit 1
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

api_pid() {
	pidfile_pid "$PIDFILE"
}

stop_api() {
	local pid
	if pid="$(api_pid)"; then
		kill_pid_wait "$pid"
		rm -f "$PIDFILE"
	fi
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
