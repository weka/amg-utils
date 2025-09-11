# CW: AMG PoC Pod Guide

## Prerequisites

- Install the AMG Helm Chart v0.1.0 with your `HF_TOKEN` - accepting all the defaults this should look something like this:
    
    ```
    $ helm install amg-cw oci://ghcr.io/sdimitro/amg-cw-chart --version 0.1.0 \
      --set wekaAmg.env.hfToken="<HF_TOKEN here - e.g. hf_1234567890abcdef>"
    ```
    
- Ensure that your HF Account has signed all the agreements for the models you want to use

## Contents of AMG PoC Pod

- NVIDIA/CUDA Runtime
- A `uv` environment under `~/amg_stable/.venv/` - the environment should be active by default when you login in the pod. The environment comes with a pre-install version of vLLM and LMCache that are known to work together (see section later about installing different versions).
- A utility called `amgctl` under your path that can be used to manage the vLLM+LMCache installation, and spawn vLLM processes.
- A few common Linux utils - `iproute2`, `vim`, etc..

## Launching vLLM Processes

You should be able to launch `vllm serve` directly from the `amg` `uv` environment and pass it any options you want. That said, the default vLLM configuration does not integrate with LMCache nor WekaFS. For this reason we’ve provided a thin wrapper on top of `vllm serve` through our `amgctl` utility that can launch vLLM with a proper configuration with a single command:

```
# amgctl host launch meta-llama/Llama-3.3-70B-Instruct
```

By default this will launch a vLLM process that uses all the GPUs on the system and enables LMCache and its WekaGDS backend. If you are curious to see all the options we pass to `vllm serve` you can add the `--dry-run` argument in the above command to list them like so:

```
# amgctl host launch meta-llama/Llama-3.3-70B-Instruct --dry-run
...<snip output>...
🔍 Dry Run Mode - vLLM Command Preview:
=====================================
Environment Variables:
  export CUFILE_ENV_PATH_JSON=/root/amg_stable/cufile.json
  export LMCACHE_WEKA_PATH=/mnt/weka/cache
  export LMCACHE_CHUNK_SIZE=256
  export LMCACHE_EXTRA_CONFIG={"gds_io_threads": 32}
  export LMCACHE_CUFILE_BUFFER_SIZE=8192
  export LMCACHE_LOCAL_CPU=false
  export LMCACHE_SAVE_DECODE_CACHE=true
  export HF_HOME=/mnt/weka/hf_cache
  export PROMETHEUS_MULTIPROC_DIR=/tmp/lmcache_prometheus
  export USE_FASTSAFETENSOR=true

Command: uv \
  run \
  vllm \
  serve \
  meta-llama/Llama-3.3-70B-Instruct \
  --tensor-parallel-size \
  8 \
  --gpu-memory-utilization \
  0.80 \
  --max-num-seqs \
  256 \
  --max-model-len \
  16384 \
  --max-num-batched-tokens \
  16384 \
  --port 
  8000 \
  --host \
  0.0.0.0 \
  --kv-transfer-config \
  {"kv_connector":"LMCacheConnectorV1","kv_role":"kv_both","kv_connector_extra_config": {}} \
  --load-format \
  fastsafetensors
```

The default configuration above are generally for sanity check and can be changed based on the goal of the PoC. For example, to benchmark the pure Weka Backend and avoid prefix caching you can add `--no-enable-prefix-caching` in the `amgctl host launch` invocation and/or increase the `max-model-len` / `max_num_batched_tokens` /…etc through `--max-model-len` / `--max-num-batched-tokens` / …etc. You can see all the configuration that can be passed with by looking at the help message aka running `amgctl host launch -h` . Two important options are `--vllm-arg` and `--vllm-env` that can pass direct strings as arguments and environment variables to `vllm serve`.

## Using different `vllm` and `LMCache` versions

`amgctl` makes it easy to change the default installation of `vLLM` and `LMCache` in the container but it only allows moving between official Github/PyPi releases for now. To move to a different version, say `vllm` `0.10.1` and `LMCache` `0.3.5` you can do the following:

1. Deactivate and uninstall the current `uv` environment with `amgctl`:
    
    ```
    # cd ~
    # deactivate
    # amgctl host clear
    ```
    
2. Install the versions of `vllm` and `lmcache` that you want:
    
    ```
    # amgctl host setup --lmcache-repo https://github.com/LMCache/LMCache.git --lmcache-branch v0.3.3 --vllm-version 0.10.1
    ```
    
3. Activate the new uv environment
    
    ```
    # source ~/amg_stable/.venv/bin/activate
    ```
