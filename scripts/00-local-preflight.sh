#!/usr/bin/env bash

echo "Checking local tools on ubuntu desktop..."
echo "Checks: ssh, scp, curl, git."

tools=("ssh" "scp" "curl" "git")

for tool in "${tools[@]}"; do
  if command -v "$tool" >/dev/null 2>&1; then
    echo "OK: $tool found"
  else
    echo "MISSING: $tool"
  fi
done

echo "Local preflight complete."
