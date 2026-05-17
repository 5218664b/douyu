#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"

echo "请选择要执行的功能："
echo "1. 一键扫码并启动树莓派推流"
echo "2. 仅更新并重启树莓派 app"
echo "3. 仅更新并重启树莓派 relay"
echo "4. 发布 app/relay/scan 镜像到树莓派"
printf "请输入编号 [1-4]: "
read -r choice

case "${choice}" in
  1)
    exec "${ROOT_DIR}/scripts/dev/scan_start_pi.sh"
    ;;
  2)
    exec "${ROOT_DIR}/scripts/dev/push_restart_app.sh"
    ;;
  3)
    exec "${ROOT_DIR}/scripts/dev/push_restart_relay.sh"
    ;;
  4)
    exec "${ROOT_DIR}/scripts/dev/release_pi_images.sh"
    ;;
  *)
    echo "无效输入: ${choice}" >&2
    exit 1
    ;;
esac
