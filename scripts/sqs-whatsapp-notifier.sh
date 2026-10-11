#!/usr/bin/env bash
# Polls the SQS queue subscribed to match-notification SNS topic and forwards events to WhatsApp via wacli.
#
# Required environment:
#   QUEUE_URL    SQS queue URL from terraform output queue_urls, usually generic-wacli
#   WACLI_TO     WhatsApp recipient: JID, phone number, or contact/group name
# Optional environment:
#   AWS_PROFILE  AWS CLI profile with sqs:ReceiveMessage/DeleteMessage (default: default)
#   AWS_REGION   Region override (default: from the profile)
#   WACLI_BIN    Path to wacli (default: ~/go/bin/wacli)
#   WACLI_ACCOUNT  Named wacli account (default: wacli's default account)
#   AWS_BIN      Path to aws cli (default: aws)
#   SLACK_WEBHOOK_URL  Slack incoming webhook notified on SQS or wacli failures
#   STATE_DIR    Persistent delivery receipts (default: ~/.local/state/sqs-whatsapp-notifier)
#
# Modes:
#   (none)              poll forever (SQS long polling, 20s)
#   --dry-run           poll and print messages without sending or deleting
#   --fixture FILE      render the SQS message body in FILE and exit (testing)
set -euo pipefail
umask 077

AWS_PROFILE="${AWS_PROFILE:-default}"
AWS_BIN="${AWS_BIN:-aws}"
QUEUE_URL="${QUEUE_URL:-}"
WACLI_TO="${WACLI_TO:-}"
WACLI_BIN="${WACLI_BIN:-$HOME/go/bin/wacli}"
WACLI_ACCOUNT="${WACLI_ACCOUNT:-}"
SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
STATE_DIR="${STATE_DIR:-$HOME/.local/state/sqs-whatsapp-notifier}"
WAIT_TIME_SECONDS=20
MAX_MESSAGES=10

log() { printf '%s %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2; }

die() { log "ERROR: $*"; exit 1; }

need_cmd() { command -v "$1" >/dev/null 2>&1 || die "missing dependency: $1"; }

notify_slack() {
	[ -z "$SLACK_WEBHOOK_URL" ] && return
	local payload
	payload="$(jq -nc --arg text "sqs-whatsapp-notifier: $*" '{text: $text}')"
	curl -fsS --max-time 10 -H 'Content-Type: application/json' --data "$payload" "$SLACK_WEBHOOK_URL" >/dev/null ||
		log "Slack notification failed"
}

format_date() {
	TZ=Europe/Zurich LC_TIME=fr_FR.UTF-8 date -d "$1" '+%a %d.%m.%Y %Hh%M' 2>/dev/null || printf '%s' "$1"
}

change_line() {
	local field="$1" old="$2" new="$3"
	case "$field" in
	playDate) printf '• 📅 Date : %s → %s' "$(format_date "$old")" "$(format_date "$new")" ;;
	hall) printf '• 🏟️ Salle : %s → %s' "$old" "$new" ;;
	homeTeam) printf '• 🔄 Domicile : %s → %s' "$old" "$new" ;;
	awayTeam) printf '• 🔄 Extérieur : %s → %s' "$old" "$new" ;;
	status) printf '• ⚠️ Statut : %s → %s' "$old" "$new" ;;
	*) printf '• ℹ️ %s : %s → %s' "$field" "$old" "$new" ;;
	esac
}

# Renders one WhatsApp message per game on stdout, games separated by
# a form-feed character so the caller can split without losing newlines.
format_messages() {
	local body="$1"
	# Validate before rendering: command/process substitutions hide failures from
	# the caller, and no malformed or unsupported event may be acknowledged.
	jq -e '
		.version == 1 and
		(.type == "volley.matches.changed" or .type == "volley.matches.new") and
		((has("eventId") | not) or (.eventId | type == "string" and length > 0)) and
		(.games | type == "array" and length > 0) and
		([.games[].gameId] | length == (unique | length)) and
		all(.games[];
			(.gameId | type == "number" and . > 0 and floor == .) and
			all(.playDate, .homeTeam, .awayTeam, .league, .hall; type == "string") and
			([.. | strings] | all(index("\u000c") == null))) and
		(if .type == "volley.matches.changed" then
			all(.games[]; (.changes | type == "array" and length > 0) and
				all(.changes[]; all(.field, .old, .new; type == "string")))
		else true end)
	' <<<"$body" >/dev/null || return 1
	case "$(jq -r '.type // ""' <<<"$body")" in
	volley.matches.changed)
		jq -c '.games[]' <<<"$body" | while IFS= read -r game; do
			{
				printf '🏐 *Match modifié : %s*\n' "$(jq -r '.league' <<<"$game")"
				printf '⚔️ %s vs %s\n' "$(jq -r '.homeTeam' <<<"$game")" "$(jq -r '.awayTeam' <<<"$game")"
				printf '📅 %s\n' "$(format_date "$(jq -r '.playDate' <<<"$game")")"
				printf '📍 %s\n' "$(jq -r '.hall' <<<"$game")"
				printf '\n'
				jq -r '.changes[] | [.field, .old, .new] | @tsv' <<<"$game" |
					while IFS=$'\t' read -r field old new; do
						change_line "$field" "$old" "$new"
						printf '\n'
					done
				printf '\f'
			}
		done
		;;
	volley.matches.new)
		jq -c '.games[]' <<<"$body" | while IFS= read -r game; do
			{
				printf '🏐 *Nouveau match : %s*\n' "$(jq -r '.league' <<<"$game")"
				printf '⚔️ %s vs %s\n' "$(jq -r '.homeTeam' <<<"$game")" "$(jq -r '.awayTeam' <<<"$game")"
				printf '📅 %s\n' "$(format_date "$(jq -r '.playDate' <<<"$game")")"
				printf '📍 %s\n' "$(jq -r '.hall' <<<"$game")"
				printf '🔖 Match #%s\n' "$(jq -r '.gameId' <<<"$game")"
				printf '\f'
			}
		done
		;;
	esac
}

send_whatsapp() {
	local text="$1"
	local args=(send text --to "$WACLI_TO" --message "$text")
	[ -n "$WACLI_ACCOUNT" ] && args+=(--account "$WACLI_ACCOUNT")
	"$WACLI_BIN" "${args[@]}" >/dev/null
}

DRY_RUN=0
FIXTURE=""
while [ $# -gt 0 ]; do
	case "$1" in
	--dry-run) DRY_RUN=1 ;;
	--fixture)
		FIXTURE="${2:-}"
		[ -n "$FIXTURE" ] || die "--fixture needs a file"
		shift
		;;
	-h | --help)
		sed -n '2,20p' "$0"
		exit 0
		;;
	*) die "unknown argument: $1" ;;
	esac
	shift
done

need_cmd jq
[ -z "$SLACK_WEBHOOK_URL" ] || need_cmd curl

if [ -n "$FIXTURE" ]; then
	[ -r "$FIXTURE" ] || die "cannot read fixture: $FIXTURE"
	format_messages "$(cat "$FIXTURE")" | while IFS= read -r -d $'\f' message; do
		[ -n "$message" ] && printf '%s\n---\n' "$message"
	done
	exit 0
fi

need_cmd "$AWS_BIN"
need_cmd "$WACLI_BIN"
need_cmd sha256sum
need_cmd flock
[ -n "$QUEUE_URL" ] || die "QUEUE_URL is not set"
[ -n "$WACLI_TO" ] || die "WACLI_TO is not set"
mkdir -p "$STATE_DIR" || die "cannot create delivery state directory"

# Single instance: a second run exits instead of duplicating notifications.
exec 9>"${TMPDIR:-/tmp}/sqs-whatsapp-notifier.lock"
flock -n 9 || die "another instance is already running"

log "polling $QUEUE_URL (profile=$AWS_PROFILE, to=$WACLI_TO)"

while true; do
	aws_args=(sqs receive-message --profile "$AWS_PROFILE" --queue-url "$QUEUE_URL"
		--max-number-of-messages "$MAX_MESSAGES" --wait-time-seconds "$WAIT_TIME_SECONDS"
		--output json)
	[ -n "${AWS_REGION:-}" ] && aws_args+=(--region "$AWS_REGION")

	if ! response="$("$AWS_BIN" "${aws_args[@]}")"; then
		log "receive-message failed; retrying in 30s"
		notify_slack "SQS receive-message failed; retrying in 30s"
		sleep 30
		continue
	fi

	# An empty body is a normal "no messages" answer; poll again quietly.
	[ -z "${response//[[:space:]]/}" ] && continue

	if ! message_count="$(jq -r '(.Messages // []) | length' <<<"$response" 2>/dev/null)" ||
		! [[ "$message_count" =~ ^[0-9]+$ ]]; then
		log "unexpected receive-message output; first bytes: [$(printf '%s' "$response" | head -c 300)]; retrying in 30s"
		notify_slack "SQS returned unexpected receive-message output; retrying in 30s"
		sleep 30
		continue
	fi
	[ "$message_count" -eq 0 ] && continue
	log "received $message_count message(s)"

	jq -c '.Messages[]' <<<"$response" | while IFS= read -r sqs_message; do
		body="$(jq -r '.Body' <<<"$sqs_message")"
		receipt_handle="$(jq -r '.ReceiptHandle' <<<"$sqs_message")"
		if ! rendered="$(format_messages "$body")" || [ -z "$rendered" ]; then
			log "invalid notification; message kept in queue"
			notify_slack "Invalid notification; message kept in queue for dead-letter handling"
			continue
		fi

		if [ "$DRY_RUN" -eq 1 ]; then
			while IFS= read -r -d $'\f' message; do
				[ -n "$message" ] && printf '%s\n---\n' "$message"
			done <<<"$rendered"
			continue
		fi
		event_id="$(jq -r '.eventId // empty' <<<"$body")"
		[ -n "$event_id" ] || event_id="$(jq -r '.MessageId // empty' <<<"$sqs_message")"
		if [ -z "$event_id" ]; then
			log "notification has no eventId or SQS MessageId; message kept in queue"
			notify_slack "Notification has no delivery identifier; message kept in queue"
			continue
		fi
		mapfile -t game_ids < <(jq -r '.games[].gameId' <<<"$body")

		send_failed=0
		game_index=0
		while IFS= read -r -d $'\f' message; do
			[ -z "$message" ] && continue
			delivery_id="$(printf '%s\n' "$QUEUE_URL" "$WACLI_ACCOUNT" "$WACLI_TO" "$event_id" "${game_ids[$game_index]}" | sha256sum)"
			delivery_file="$STATE_DIR/${delivery_id%% *}"
			game_index=$((game_index + 1))
			[ -f "$delivery_file" ] && continue
			if ! send_whatsapp "$message"; then
				log "wacli send failed; message kept in queue"
				notify_slack "wacli send failed; message kept in queue"
				send_failed=1
			elif ! touch "$delivery_file"; then
				log "cannot persist delivery receipt; message kept in queue"
				notify_slack "Cannot persist delivery receipt; message may be delivered again"
				send_failed=1
			fi
		done <<<"$rendered"

		if [ "$send_failed" -eq 0 ]; then
			delete_args=(sqs delete-message --profile "$AWS_PROFILE" --queue-url "$QUEUE_URL"
				--receipt-handle "$receipt_handle")
			[ -n "${AWS_REGION:-}" ] && delete_args+=(--region "$AWS_REGION")
			if ! "$AWS_BIN" "${delete_args[@]}" >/dev/null; then
				log "delete-message failed; message may be redelivered"
				notify_slack "SQS delete-message failed; message may be redelivered"
			fi
		fi
	done
done
