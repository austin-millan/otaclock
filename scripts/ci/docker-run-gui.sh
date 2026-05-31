#!/usr/bin/env bash
# Run an OtaClock2 image against the host X11 display, forwarding API env vars.
set -euo pipefail

IMAGE="${1:?docker image reference}"

if [[ -z "${DISPLAY:-}" ]]; then
  echo "DISPLAY is not set — start an X11 or XWayland session first." >&2
  exit 1
fi

xhost +local:docker >/dev/null 2>&1 || true

api_env=()
for v in ANTHROPIC_API_KEY OPENAI_API_KEY OTA_LLM_PROVIDER OTA_24H_CLOCK ANTHROPIC_MODEL OPENAI_MODEL; do
  eval "val=\${$v:-}"
  [[ -n "$val" ]] && api_env+=(-e "$v=$val")
done

x11_mounts=(-v /tmp/.X11-unix:/tmp/.X11-unix:rw)
if [[ -n "${XAUTHORITY:-}" && -f "${XAUTHORITY}" ]]; then
  x11_mounts+=(-e "XAUTHORITY=${XAUTHORITY}" -v "${XAUTHORITY}:${XAUTHORITY}:ro")
fi

# Forward the sound device when present; otherwise the widget runs without audio.
sound_args=()
if [[ -d /dev/snd ]]; then
  sound_args+=(--device /dev/snd)
fi

# Forward the GPU for hardware-accelerated GL when present. Without it Mesa
# falls back to software rendering (llvmpipe) and prints harmless DRI warnings,
# so force software rendering to keep the output clean.
render_args=()
if [[ -d /dev/dri ]]; then
  render_args+=(--device /dev/dri)
else
  render_args+=(-e LIBGL_ALWAYS_SOFTWARE=1)
fi

docker run --rm -it \
  --name otaclock2 \
  -e DISPLAY="${DISPLAY}" \
  "${x11_mounts[@]}" \
  "${sound_args[@]}" \
  "${render_args[@]}" \
  "${api_env[@]}" \
  "${IMAGE}"
