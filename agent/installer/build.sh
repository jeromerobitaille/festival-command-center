#!/usr/bin/env bash
# Construit Setup.exe à partir des binaires produits par GoReleaser. Usage : installer/build.sh <version>
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION=${1:?version}
AGENT=$(ls dist/agent_windows_amd64*/agent.exe | head -1)
TRAY=$(ls dist/agent-tray_windows_amd64*/agent-tray.exe | head -1)
mkdir -p dist/installer
makensis -V2 -DVERSION="$VERSION" -DAGENT="$PWD/$AGENT" -DTRAY="$PWD/$TRAY" -DOUT="$PWD/dist/installer/FestivalCommandCenter-Agent-Setup-$VERSION.exe" installer/festival-agent.nsi
ls -la dist/installer
