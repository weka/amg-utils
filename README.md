# AMG Utils

## Installing `amgctl`

### Prerequisites
Install uv with:
```bash
wget -qO- https://astral.sh/uv/install.sh | sh
```

### Download and Install
1. Download the utility from the releases:
```bash
wget https://github.com/weka/amg-utils/releases/latest/download/amgctl-linux-amd64 -O amgctl
```

2. Make it executable:
```bash
chmod +x amgctl
```

3. Optionally, put it in your `$PATH`:
```bash
sudo mv amgctl /usr/local/bin/
```

4. To upgrade the version, just run:
```bash
amgctl update
```

## Docker Images

Docker images are built manually using the provided scripts in the `docker/scripts/` directory:

- **Local Development**: `./docker/scripts/build-docker-local.sh` - Build images locally for testing
- **Production Release**: `./docker/scripts/build-and-push-docker.sh` - Build and push to Docker Hub

See [`docker/scripts/README.md`](docker/scripts/README.md) for detailed usage instructions.

