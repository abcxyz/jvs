#!/usr/bin/env bash

set -eEuo pipefail

PLUGIN_DIR_SRC=plugin-dir-src
PLUGIN_VERSION='0.0.2-rc'
PLUGIN_NAME=jvs-plugin-jira

for goos in 'amd64' 'arm64'
do
    file_name=jvs-plugin-jira_${PLUGIN_VERSION}_darwin_${goos}.tar.gz
    plugin_url=https://github.com/abcxyz/jvs-plugin-jira/releases/download/v${PLUGIN_VERSION}/${file_name}
    wget ${plugin_url}
    mkdir -p tmp/${goos}
    tar xzfC ${file_name} tmp/${goos}
    mkdir -p ${PLUGIN_DIR_SRC}-$goos
    mv tmp/$goos/$PLUGIN_NAME $PLUGIN_DIR_SRC-$goos/$PLUGIN_NAME
    rm ${file_name}
done 
rm -r tmp
