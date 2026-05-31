FROM mcr.microsoft.com/devcontainers/go:1-bookworm

# Remove broken Yarn repository to prevent apt-get update failures
RUN rm -f /etc/apt/sources.list.d/yarn.list && \
    rm -f /etc/apt/sources.list.d/yarnpkg.list
