# The folder holding the plugins from downloadPlugin.sh
ARG PLUGIN_DIR_SRC
# The folder should be consistent with JustificationConfig.PluginDir
ARG PLUGIN_DIR_DEST
# The jvs image.
ARG JVS_IMAGE

FROM ${JVS_IMAGE}

COPY ${PLUGIN_DIR_SRC}/* ${PLUGIN_DIR_DEST}/

# Normally we would set this to run as "nobody". But goreleaser builds the
# binary locally and sometimes it will mess up the permission and cause "exec
# user process caused: permission denied".
#
# USER nobody

# Run the CLI
ENTRYPOINT ["/bin/jvsctl"]
