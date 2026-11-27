#!/bin/sh
# workspace-diff.sh: show what `gtm-cli versions create` would put in a version.
# Each entity the workspace adds, updates or deletes is printed as a unified diff
# against the same entity in the latest container version.
#
# IDs come from GTM_ACCOUNT_ID, GTM_CONTAINER_ID and GTM_WORKSPACE_ID, or from
# the gtm-cli config file. Needs gtm-cli, jq and diff.
# Exit status: 0 no differences, 1 differences, anything else an error.
#
# The comparison is with the latest version, which a workspace is based on, not
# with the live one. If GTM says the workspace is out of date, update it first.
set -eu

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

gtm-cli workspaces status --entities >"$tmp/status.json"
if [ "$(jq '.status.hasConflicts' "$tmp/status.json")" = true ]; then
	echo "workspace-diff: the workspace has merge conflicts; resolve them in GTM first" >&2
	exit 2
fi
gtm-cli versions get --latest >"$tmp/base.json"

# Pair each change with the same entity in the version, dropping the fields that
# always differ between a workspace and a version.
jq --slurpfile base "$tmp/base.json" '
	def idkey: {tag: "tagId", trigger: "triggerId", variable: "variableId",
		folder: "folderId", builtInVariable: "type", customTemplate: "templateId",
		client: "clientId", transformation: "transformationId", zone: "zoneId",
		gtagConfig: "gtagConfigId"}[.] // "";
	def clean: if . == null then null
		else del(.path, .fingerprint, .tagManagerUrl, .workspaceId) end;
	$base[0].version as $v
	| [.status.changes[] | .type as $t | .id as $id | {
		label: ("\($t)/\($id)" + if .name then " " + (.name | tojson) else "" end),
		status: .changeStatus,
		before: ([$v[$t][]? | select(.[$t | idkey] == $id)] | first | clean),
		after: (if .changeStatus == "deleted" then null else .entity | clean end)
	}]' "$tmp/status.json" >"$tmp/pairs.json"

result=0
count=$(jq length "$tmp/pairs.json")
i=0
while [ "$i" -lt "$count" ]; do
	label=$(jq -r ".[$i].label" "$tmp/pairs.json")
	change=$(jq -r ".[$i].status" "$tmp/pairs.json")
	jq -S ".[$i].before" "$tmp/pairs.json" >"$tmp/before.json"
	jq -S ".[$i].after" "$tmp/pairs.json" >"$tmp/after.json"
	diff -u -L "$label in the latest version" -L "$label in the workspace ($change)" \
		"$tmp/before.json" "$tmp/after.json" || {
		code=$?
		[ "$code" -eq 1 ] || exit "$code"
		result=1
	}
	i=$((i + 1))
done
exit "$result"
