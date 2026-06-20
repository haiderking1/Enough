#!/usr/bin/env bash
# Print the next pending leaf package path from port-manifest.json.
# A leaf package is pending and has no remaining pending internal dependencies.
set -euo pipefail

MANIFEST="${1:-port-manifest.json}"

python3 - "$MANIFEST" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    manifest = json.load(f)

packages = manifest.get("packages", [])
pending = {p["path"] for p in packages if p.get("status") == "pending"}
if not pending:
    print("NO_PENDING_PACKAGES")
    sys.exit(0)

leaf = None
for p in packages:
    if p.get("status") != "pending":
        continue
    if not any(dep in pending for dep in p.get("depends_on", [])):
        leaf = p["path"]
        break

if leaf:
    print(leaf)
else:
    # No pure leaf left; return the first pending package in manifest order.
    print(next(iter(pending)))
PY
