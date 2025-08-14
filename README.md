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
wget https://github.com/weka/amg-scripts/releases/download/v0.1.4/amgctl-linux-amd64 -O amgctl
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
