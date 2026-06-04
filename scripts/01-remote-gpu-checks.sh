#!/usr/bin/env bash
set -u

echo "Checking remote GPU Droplet..."
echo "This script should be run on the AMD GPU Droplet, not on the local laptop."

echo
echo "Hostname:"
hostname

echo
echo "OS details:"
cat /etc/os-release

echo
echo "Checking Bash version"
if (( BASH_VERSINFO[0] < 4 )); then
  echo "ERROR: This script requires Bash 4.0 or newer."
  exit 1
else
  echo "Bash version can run this script!"
fi

echo
echo "Checking required commands for ROCm and Docker..."

tools=("rocminfo" "rocm-smi" "docker")

declare -A usability_checks
usability_checks["docker"]="docker ps"

for tool in "${tools[@]}"; do
  if command -v "$tool" >/dev/null 2>&1; then
    echo "OK: $tool found"

    if [[ -n "${usability_checks[$tool]+exists}" ]]; then
      echo "Checking whether $tool is usable..."

      if ${usability_checks[$tool]} >/dev/null 2>&1; then
        echo "OK: $tool usability check passed"
      else
        echo "WARNING: $tool exists, but usability check failed"
        echo "Command failed: ${usability_checks[$tool]}"
      fi
    fi
  else
    echo "MISSING: $tool"
  fi
done

echo
echo "Checking AMD GPU device files..."

device_files=("/dev/kfd" "/dev/dri")

for device_file in "${device_files[@]}"; do
  if [ -e "$device_file" ]; then
    echo "OK: $device_file exists"
  else
    echo "MISSING: $device_file"
  fi
done

echo
echo "Running rocm-smi if available..."
if command -v rocm-smi >/dev/null 2>&1; then
  rocm-smi
else
  echo "Skipping rocm-smi because it is not installed."
fi

echo
echo "Running rocminfo summary if available..."
if command -v rocminfo >/dev/null 2>&1; then
  rocminfo | grep -E "Name:|Marketing Name:|Device Type:" | head -80
else
  echo "Skipping rocminfo because it is not installed."
fi

echo
echo "Checking Docker version..."
if command -v docker >/dev/null 2>&1; then
  docker --version
else
  echo "Skipping Docker version because docker is not installed."
fi

echo
echo "Interpretation:"
echo "- If this was run on your local laptop, missing ROCm tools may be OK."
echo "- If this was run on the AMD GPU Droplet, rocminfo and rocm-smi should normally be present."
echo "- On the AMD GPU Droplet, Docker should also be available before running vLLM."
echo
echo "Remote GPU checks complete."
