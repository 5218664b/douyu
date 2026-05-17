#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"

echo "请选择要执行的功能："
echo "1. 一键扫码并启动树莓派推流"
echo "2. 仅更新并重启树莓派 app"
echo "3. 仅更新并重启树莓派 relay"
echo "4. 仅更新并重启树莓派 scan-provider"
echo "5. 发布 app 镜像到树莓派"
echo "6. 发布 relay 镜像到树莓派"
echo "7. 发布 scan-provider 镜像到树莓派"
echo "8. 发布 app/relay/scan 全部镜像到树莓派"
echo "9. 远程一键重新部署树莓派 app + relay"
echo "10. 仅用 compose 重建树莓派 scan-provider"
printf "请输入编号 [1-10]: "
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
    exec "${ROOT_DIR}/scripts/dev/push_restart_scan_provider.sh"
    ;;
  5)
    exec "${ROOT_DIR}/scripts/dev/push_pi_app_image.sh"
    ;;
  6)
    exec "${ROOT_DIR}/scripts/dev/push_pi_relay_image.sh"
    ;;
  7)
    exec "${ROOT_DIR}/scripts/dev/push_pi_scan_image.sh"
    ;;
  8)
    exec "${ROOT_DIR}/scripts/dev/release_pi_images.sh"
    ;;
  9)
    exec "${ROOT_DIR}/scripts/dev/redeploy_pi.sh"
    ;;
  10)
    exec "${ROOT_DIR}/scripts/dev/push_restart_scan_provider.sh"
    ;;
  *)
    echo "无效输入: ${choice}" >&2
    exit 1
    ;;
esac
