#!/usr/bin/env bash
# Hosted Ubuntu runners sometimes advertise a very slow Azure mirror. Keep the
# signed Ubuntu repositories, but use their official HTTPS endpoint directly.
set -euo pipefail

printf 'Acquire::Retries "2";\nAcquire::http::Timeout "20";\nAcquire::https::Timeout "20";\n' |
    sudo tee /etc/apt/apt.conf.d/99flow-ci-timeouts >/dev/null
for file in /etc/apt/apt-mirrors.txt /etc/apt/sources.list /etc/apt/sources.list.d/ubuntu.sources; do
    if [ -f "$file" ]; then
        sudo sed -i 's#https\?://azure.archive.ubuntu.com/ubuntu#https://archive.ubuntu.com/ubuntu#g' "$file"
    fi
done
if [ "$#" -gt 0 ]; then
    sudo apt-get update -qq
    sudo apt-get install --no-install-recommends -y "$@"
fi
