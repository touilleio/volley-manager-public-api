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
#
# Modes:
#   (none)              poll forever (SQS long polling, 20s)
#   --dry-run           poll and print messages without sending or deleting
#   --fixture FILE      render the SQS message body in FILE and exit (testing)
set -euo pipefail

AWS_PROFILE="${AWS_PROFILE:-default}"
AWS_BIN="${AWS_BIN:-aws}"
QUEUE_URL="${QUEUE_URL:-}"
WACLI_TO="${WACLI_TO:-}"
WACLI_BIN="${WACLI_BIN:-$HOME/go/bin/wacli}"
WACLI_ACCOUNT="${WACLI_ACCOUNT:-}"
WAIT_TIME_SECONDS=20
MAX_MESSAGES=10

log() { printf '%s %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2; }

die() { log "ERROR: $*"; exit 1; }

need_cmd() { command -v "$1" >/dev/null 2>&1 || die "missing dependency: $1"; }

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

if [ -n "$FIXTURE" ]; then
	[ -r "$FIXTURE" ] || die "cannot read fixture: $FIXTURE"
	format_messages "$(cat "$FIXTURE")" | while IFS= read -r -d $'\f' message; do
		[ -n "$message" ] && printf '%s\n---\n' "$message"
	done
	exit 0
fi

[ -x "$AWS_BIN" ] || die "aws cli not found or not executable: $AWS_BIN"
[ -x "$WACLI_BIN" ] || die "wacli not found or not executable: $WACLI_BIN"
[ -n "$QUEUE_URL" ] || die "QUEUE_URL is not set"
[ -n "$WACLI_TO" ] || die "WACLI_TO is not set"

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
		sleep 30
		continue
	fi

	# An empty body is a normal "no messages" answer; poll again quietly.
	[ -z "${response//[[:space:]]/}" ] && continue

	if ! message_count="$(jq -r '(.Messages // []) | length' <<<"$response" 2>/dev/null)" ||
		! [[ "$message_count" =~ ^[0-9]+$ ]]; then
		log "unexpected receive-message output; first bytes: [$(printf '%s' "$response" | head -c 300)]; retrying in 30s"
		sleep 30
		continue
	fi
	[ "$message_count" -eq 0 ] && continue
	log "received $message_count message(s)"

	jq -c '.Messages[]' <<<"$response" | while IFS= read -r sqs_message; do
		body="$(jq -r '.Body' <<<"$sqs_message")"
		receipt_handle="$(jq -r '.ReceiptHandle' <<<"$sqs_message")"

		if [ "$DRY_RUN" -eq 1 ]; then
			format_messages "$body" | while IFS= read -r -d $'\f' message; do
				[ -n "$message" ] && printf '%s\n---\n' "$message"
			done
			continue
		fi

		send_failed=0
		while IFS= read -r -d $'\f' message; do
			[ -z "$message" ] && continue
			if ! send_whatsapp "$message"; then
				log "wacli send failed; message kept in queue"
				send_failed=1
			fi
		done < <(format_messages "$body")

		if [ "$send_failed" -eq 0 ]; then
			delete_args=(sqs delete-message --profile "$AWS_PROFILE" --queue-url "$QUEUE_URL"
				--receipt-handle "$receipt_handle")
			[ -n "${AWS_REGION:-}" ] && delete_args+=(--region "$AWS_REGION")
			"$AWS_BIN" "${delete_args[@]}" >/dev/null || log "delete-message failed; message may be redelivered"
		fi
	done
done
