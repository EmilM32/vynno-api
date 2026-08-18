# Shared helpers for scripts/ in this repo. Source from those scripts only.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PIDFILE="$ROOT/var/api.pid"
COMPOSE=(docker compose --project-directory "$ROOT")

die() {
	echo "$*" >&2
	exit 1
}

require_env_file() {
	if [[ ! -f "$ROOT/.env" ]]; then
		die "missing $ROOT/.env — copy .env.example and set BOOTSTRAP_PASSWORD"
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

ensure_postgres() {
	"${COMPOSE[@]}" up -d
	wait_for_postgres
}
