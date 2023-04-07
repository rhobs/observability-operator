header() {
	local title="🔆🔆🔆  $*  🔆🔆🔆 "

	local len=40
	if [[ ${#title} -gt $len ]]; then
		len=${#title}
	fi

	echo -e "\n\n  \033[1m${title}\033[0m"
	echo -n "━━━━━"
	printf '━%.0s' $(seq "$len")
	echo "━━━━━━━"

}

info() {
	echo " 🔔 $*"
}

ok() {
	echo " ✅ $*"
}

warn() {
	echo " ⚠️  $*"
}

skip() {
	echo " 🙈 SKIP: $*"
}

die() {
	echo -e "\n ✋ $* "
	echo -e "──────────────────── ⛔️⛔️⛔️ ────────────────────────\n"
	exit 1
}

line() {
	local len="$1"
	shift

	echo -n "────"
	printf '─%.0s' $(seq "$len")
	echo "────────"
}

# wait_for_operators_ready requires the namespace where the operator is installed
wait_for_operators_ready() {
	local ns="$1"
	shift

	header "Wait for ObO to be Ready"

	local tries=30
	while [[ $tries -gt 0 ]] &&
		! kubectl -n "$ns" rollout status deploy/obo-prometheus-operator; do
		sleep 10
		((tries--))
	done

	kubectl wait -n "$ns" --for=condition=Available deploy/obo-prometheus-operator --timeout=300s
	kubectl wait -n "$ns" --for=condition=Available deploy/obo-prometheus-operator-admission-webhook --timeout=300s
	kubectl wait -n "$ns" --for=condition=Available deploy/observability-operator --timeout=300s

	ok "Obo up and running"
}
