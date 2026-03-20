FROM timescale/timescaledb:latest-pg16

USER root

# Install build dependencies (Alpine)
RUN apk add --no-cache \
    build-base \
    curl \
    git \
    postgresql-dev \
    pkgconfig \
    openssl-dev \
    clang

# Build and install pgvector from source
RUN cd /tmp && \
    git clone --branch v0.8.0 --depth 1 https://github.com/pgvector/pgvector.git && \
    cd pgvector && \
    make && \
    make install && \
    rm -rf /tmp/pgvector

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install cargo-pgrx and build PGMQ
RUN cargo install cargo-pgrx@0.11.3 --locked \
    && cargo pgrx init --pg16 $(pg_config --pgxs | sed 's/pgxs//g') \
    && cd /tmp \
    && git clone --branch v1.4.4 --depth 1 https://github.com/tembo-io/pgmq.git \
    && cd pgmq \
    && cargo pgrx install --pg-config $(pg_config) \
    && cd / \
    && rm -rf /tmp/pgmq /root/.cargo/registry /root/.cargo/git

USER postgres

# Copy initialization script
COPY scripts/init-extensions.sql /docker-entrypoint-initdb.d/

EXPOSE 5432