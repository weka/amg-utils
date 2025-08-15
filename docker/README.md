## Docker Weka AMG

Build the image:
```
docker build -t <..pick name..> .
```

Run it in a daemon:
```
docker run -d \
	--gpus <..> \
	--runtime nvidia \
	--network <..pick net..> \
	--ipc host \
	--device <..device you want to share..> \
	-v /etc/cufile.json:/etc/cufile.json:ro \
	-v <..weka path..>:<..weka path..> \
	--name <..> \
	<..image name from build command..> \
	tail -f /dev/null
```

## Example Commands

Assumptions:
* I want to give all the GPUs in the system to the container
* I want to share all the host's network interfaces with the container (no Docker NAT which is better for HPC/AI)
* I want to share the whole IPC namespace of the host with the container (can share memory segments for GDS)
* I want to use 8 RDMA devices through user-space verbs interfaces
* My weka mount is under `/mnt/weka`

Example build command:
```
docker build -t ${USER}-amg .
```

To run it in a daemon:
```
docker run -d \
        --gpus all \
        --runtime nvidia \
        --network host \
        --ipc host \
	--device=/dev/infiniband/rdma_cm \
	--device=/dev/infiniband/uverbs0 \
	--device=/dev/infiniband/uverbs1 \
	--device=/dev/infiniband/uverbs2 \
	--device=/dev/infiniband/uverbs3 \
	--device=/dev/infiniband/uverbs4 \
	--device=/dev/infiniband/uverbs5 \
	--device=/dev/infiniband/uverbs6 \
	--device=/dev/infiniband/uverbs7 \
	--device=/dev/infiniband/uverbs8 \
	--device=/dev/infiniband/uverbs9 \
	--device=/dev/infiniband/uverbs10 \
	--device=/dev/infiniband/uverbs11 \
        -v /etc/cufile.json:/etc/cufile.json:ro \
        -v /mnt/weka:/mnt/weka \
        --name ${USER}-amg-docker \
        ${USER}-amg \
        tail -f /dev/null
```

To connect to the container:
```
docker exec -it ${USER}-amg-docker /bin/bash
```

Other commands:
```
# Stop the container
docker stop ${USER}-amg-docker

# Restart the stopped container
docker start ${USER}-amg-docker

# Remove the container (will lose any changes made inside)
docker rm ${USER}-amg-docker
```

## Troubleshooting GDS & vLLM+LMCache

```
# Verify GPU access
$ nvidia-smi

# Check GDS installation - you should see:
# ...
# =====================
# DRIVER CONFIGURATION:
# =====================
# ..
#  NVMe               : Supported
# ..
#  WekaFS             : Supported
#  Userspace RDMA     : Supported
# --Mellanox PeerDirect : Enabled
# --rdma library        : Loaded (libcufile_rdma.so)
# --rdma devices        : Configured
# --rdma_device_status  : Up: 8 Down: 0
# ...
$ gdscheck -p

# Run GDSIO
$ gdsio <..args..>
```
Don't worry if you see messages like `get_mempolicy: Operation not permitted` or `set_mempolicy: Operation not permitted` when running GDSIO. You'll still be able to get

## Troubleshooting vLLM+LMCache

```
# Check conda environment
$ conda info --envs

# Verify LMCache installation
$ python -c "import lmcache; print('LMCache imported successfully')"

# Check vLLM installation
python -c "import vllm; print('vLLM imported successfully')"
```
