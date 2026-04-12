# syntax=docker/dockerfile:1
FROM debian:13-slim
ARG APP_NAME=averninstats

COPY dist/${APP_NAME} /usr/local/bin/${APP_NAME}
COPY <<-"EOF" /usr/local/bin/entrypoint.sh
#!/bin/sh
# =============================================================================
# entrypoint.sh – Docker Entrypoint with hook support
# =============================================================================
# Place executable scripts in /docker-entrypoint.d/ to run them before the
# main process starts. Scripts are executed in lexicographic order.
#
# Naming convention (optional but recommended):
#   /docker-entrypoint.d/10-init-config.sh
#   /docker-entrypoint.d/20-wait-for-db.sh
#   /docker-entrypoint.d/90-custom-user-script.sh
# =============================================================================

set -e

HOOK_DIR="/docker-entrypoint.d"

# -----------------------------------------------------------------------------
# Run hook scripts from $HOOK_DIR
# -----------------------------------------------------------------------------
run_hooks() {
    if [ -d "$HOOK_DIR" ]; then
        # Check if directory contains any files
        found=$(find "$HOOK_DIR" -maxdepth 1 -type f | sort)

        if [ -z "$found" ]; then
            echo "[entrypoint] No hook scripts found in $HOOK_DIR, skipping."
        else
            echo "[entrypoint] Running hook scripts from $HOOK_DIR ..."
            for script in $(find "$HOOK_DIR" -maxdepth 1 -type f | sort); do
                if [ -x "$script" ]; then
                    echo "[entrypoint] Running: $script"
                    "$script"
                else
                    echo "[entrypoint] Skipping (not executable): $script"
                fi
            done
            echo "[entrypoint] Hook scripts done."
        fi
    fi
}

run_hooks

# -----------------------------------------------------------------------------
# Hand off to the main process.
# Using "exec" replaces the shell with the process so it receives signals
# correctly (e.g. SIGTERM on `docker stop`).
# -----------------------------------------------------------------------------
exec "$@"
EOF

RUN groupadd -f nonroot \
    && useradd -g nonroot nonroot \
    && mkdir -p /docker-entrypoint.d \
    && chown -R nonroot:nonroot /usr/local/bin /docker-entrypoint.d \
    && chmod -R +x /usr/local/bin

USER nonroot

VOLUME ["/docker-entrypoint.d"]

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["/usr/local/bin/averninstats"]
