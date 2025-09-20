#!/usr/bin/env bash

# -*- indent-tabs-mode: nil; tab-width: 2; sh-indentation: 2; -*-

# Entrypoint script for the amg container - helpful because the container can start in non-GPU environments for debugging purposes

set -euo pipefail

# helper functions
log()   { printf '%s %s\n' "$(date +'%FT%T%z')" "$*" >&2; }
fatal() { printf '%s %s\n' "$(date +'%FT%T%z')" "$*" >&2; exit 90; }
have_cmd() { command -v "$1" >/dev/null 2>&1; }

is_gpu_env() {
  [ -e /dev/nvidiactl ] && return 0
  [ -e /dev/nvidia0 ] && return 0
  [ -e /proc/driver/nvidia/version ] && return 0
  have_cmd nvidia-smi && return 0
  { [ -n "${CUDA_VISIBLE_DEVICES:-}" ] && [ "${CUDA_VISIBLE_DEVICES}" != "-1" ]; } && return 0
  return 1
}

main() {
  if ! is_gpu_env; then
    log "INFO: No NVIDIA GPUs detected; skipping 'amgctl host setup'."
    exec "$@"
  fi

  log "INFO: NVIDIA/GPU environment detected; running 'amgctl host setup'."

  have_cmd amgctl || fatal "FATAL: 'amgctl' not found in PATH."

  amgctl host setup
  log "INFO: 'amgctl host setup' completed successfully."

  exec "$@"
}

main "$@"
