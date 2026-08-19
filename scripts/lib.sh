# Shared helpers for scripts/ in this repo. Source from those scripts only.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PIDFILE="$ROOT/var/api.pid"
COMPOSE=(docker compose --project-directory "$ROOT")
PROD_DB=vynno
DEV_DB=vynno_dev

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

api_pid() {
	if [[ ! -f "$PIDFILE" ]]; then
		return 1
	fi
	local pid
	pid="$(cat "$PIDFILE")"
	if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
		echo "$pid"
		return 0
	fi
	rm -f "$PIDFILE"
	return 1
}

stop_api() {
	local pid
	if pid="$(api_pid)"; then
		kill "$pid" 2>/dev/null || true
		local i
		for i in $(seq 1 20); do
			if ! kill -0 "$pid" 2>/dev/null; then
				break
			fi
			sleep 0.1
		done
		if kill -0 "$pid" 2>/dev/null; then
			kill -9 "$pid" 2>/dev/null || true
		fi
		rm -f "$PIDFILE"
	fi
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
