# Use the official PostGIS image for PostgreSQL 17
FROM postgis/postgis:17-3.5

# Combine all apt-get operations into a single RUN layer to reduce image size.
# Add ca-certificates for SSL/TLS validation and other necessary tools.
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    postgresql-server-dev-17 \
    curl \
    gnupg \
    lsb-release \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# --- pgvector Installation ---
 RUN curl -sfSL https://github.com/pgvector/pgvector/archive/refs/tags/v0.8.0.tar.gz | tar xz -C /tmp && \
    cd /tmp/pgvector-0.8.0 && make && make install && \
    rm -rf /tmp/pgvector-0.8.0

# --- TimescaleDB Installation ---
# Add TimescaleDB repository, install the extension, and clean up.
# This is now done in a separate, clean layer.
RUN apt-get update && \
    # Ensure the target directory for sources.list.d exists (robustness).
    mkdir -p /etc/apt/sources.list.d && \
    # Use -fsSL flags with curl for better error handling and to follow redirects.
    curl -fsSL https://packagecloud.io/timescale/timescaledb/gpgkey | gpg --dearmor -o /usr/share/keyrings/timescaledb.keyring && \
    echo "deb [signed-by=/usr/share/keyrings/timescaledb.keyring] https://packagecloud.io/timescale/timescaledb/debian/ $(lsb_release -cs) main" > /etc/apt/sources.list.d/timescaledb.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends timescaledb-2-postgresql-17 timescaledb-tools && \
    # Clean up apt cache to keep the image small.
    rm -rf /var/lib/apt/lists/*