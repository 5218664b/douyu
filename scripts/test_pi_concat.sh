#!/usr/bin/env sh
set -eu

MEDIA_DIR="${1:?media dir is required}"
LIST_FILE="${2:-/tmp/douyu-test.ffconcat}"

first_file="$(find "${MEDIA_DIR}" -maxdepth 1 -name '*.ts' | sort | sed -n '1p')"
second_file="$(find "${MEDIA_DIR}" -maxdepth 1 -name '*.ts' | sort | sed -n '2p')"

if [ -z "${first_file}" ] || [ -z "${second_file}" ]; then
  echo "need at least two .ts files under ${MEDIA_DIR}" >&2
  exit 1
fi

{
  printf 'ffconcat version 1.0\n'
  printf "file '%s'\n" "${first_file}"
  printf "file '%s'\n" "${second_file}"
} > "${LIST_FILE}"

echo "== ffconcat =="
cat "${LIST_FILE}"
echo

ffmpeg -hide_banner -loglevel info -f concat -safe 0 -i "${LIST_FILE}" -t 8 -c copy -f null -
