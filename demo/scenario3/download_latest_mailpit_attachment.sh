#!/usr/bin/env bash
set -euo pipefail

MAILPIT_URL="${MAILPIT_URL:-http://localhost:8025}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_FILE="${SCRIPT_DIR}/package.json"
TMP_MESSAGES="$(mktemp)"
TMP_MESSAGE_DETAIL="$(mktemp)"
trap 'rm -f "${TMP_MESSAGES}" "${TMP_MESSAGE_DETAIL}"' EXIT

curl -s "${MAILPIT_URL}/api/v1/messages" > "${TMP_MESSAGES}"

MESSAGE_ID="$(
  ruby -rjson -e '
    body = JSON.parse(File.read(ARGV[0]))
    messages = body["messages"] || body["Messages"] || []
    abort("No messages found in Mailpit") if messages.empty?
    latest = messages.first
    message_id = latest["ID"] || latest["id"]
    print message_id
  ' "${TMP_MESSAGES}"
)"

curl -s "${MAILPIT_URL}/api/v1/message/${MESSAGE_ID}" > "${TMP_MESSAGE_DETAIL}"

ATTACHMENT_ID="$(
  ruby -rjson -e '
    body = JSON.parse(File.read(ARGV[0]))
    attachments = body["Attachments"] || body["attachments"] || []
    attachment = attachments.find { |item| (item["FileName"] || item["filename"] || "").end_with?(".json") }
    abort("No JSON attachment found in latest Mailpit message") unless attachment
    print attachment["PartID"] || attachment["part_id"] || attachment["id"]
  ' "${TMP_MESSAGE_DETAIL}"
)"

curl -s "${MAILPIT_URL}/api/v1/message/${MESSAGE_ID}/part/${ATTACHMENT_ID}" > "${OUT_FILE}"
echo "Saved ${OUT_FILE}"
