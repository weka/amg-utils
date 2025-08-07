#!/usr/bin/env python3
"""
LMCache Chunk Size Profiling Benchmark

This script benchmarks LMCache retrieval performance across different chunk sizes
and token counts to identify bottlenecks and overhead sources using advanced profiling tools.

Uses pyinstrument for statistical profiling and optional line_profiler for line-by-line analysis.
These tools provide more accurate and less variant results compared to cProfile.

Model Configuration:
- Configurable number of layers (default: 2 layers, use --num-layers to adjust)
- With 2 layers: Data size per token = 2 layers * 2 (K,V) * 32 heads * 128 head_size * 2 bytes = 32KB/token
- With 32 layers: Data size per token = 32 layers * 2 (K,V) * 32 heads * 128 head_size * 2 bytes = 512KB/token
- Default token counts (256, 512, 1024, 2048) with 2 layers generate files from 8MB to 64MB
- Same token counts with 32 layers generate files from 128MB to 1GB
"""

#
# Copyright 2025 Serapheim Dimitropoulos <serapheim.dimitropoulos@weka.io>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

import argparse
import time
import functools
import os
import sys
from typing import List, Dict, Any, Optional
from dataclasses import dataclass
import json

# Misc imports
try:
    import torch
    import numpy as np
except ImportError:
    LMCACHE_AVAILABLE = False
    print("WARNING: torch or numpy not available. Are you running this from the AMG conda environment?")

# Profiling imports
try:
    import pyinstrument
    PYINSTRUMENT_AVAILABLE = True
except ImportError:
    PYINSTRUMENT_AVAILABLE = False
    print("WARNING: pyinstrument not available. Install with: pip install pyinstrument")

try:
    import line_profiler
    LINE_PROFILER_AVAILABLE = True
except ImportError:
    LINE_PROFILER_AVAILABLE = False
    print("WARNING: line_profiler not available. Install with: pip install line_profiler")

# LMCache imports
try:
    from lmcache.v1.cache_engine import LMCacheEngineBuilder
    from lmcache.v1.config import LMCacheEngineConfig
    from lmcache.v1.cache_engine import LMCacheEngineMetadata
    from lmcache.v1.gpu_connector import VLLMPagedMemGPUConnectorV2
    LMCACHE_AVAILABLE = True
except ImportError:
    LMCACHE_AVAILABLE = False
    print("WARNING: lmcache not available. Are you running this from the AMG conda environment?")


@dataclass
class ProfilingResults:
    """Container for profiling results"""
    chunk_size: int
    num_tokens: int
    num_chunks: int
    retrieve_time: float
    throughput_gbps: float
    data_size_mb: float
    detailed_timings: Dict[str, float]
    profiling_output: Optional[str] = None
    # Add error tracking
    memory_errors: int = 0
    error_details: List[str] = None

    def __post_init__(self):
        if self.error_details is None:
            self.error_details = []


class DetailedTimer:
    """Context manager for detailed timing measurements"""
    
    def __init__(self, name: str, collector: Dict[str, List[float]]):
        self.name = name
        self.collector = collector
        self.start_time = None
    
    def __enter__(self):
        torch.cuda.synchronize()
        self.start_time = time.perf_counter()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        torch.cuda.synchronize()
        elapsed = time.perf_counter() - self.start_time
        if self.name not in self.collector:
            self.collector[self.name] = []
        self.collector[self.name].append(elapsed)


class AdvancedProfiler:
    """Advanced profiling wrapper that supports multiple profiling backends"""
    
    def __init__(self, profiler_type: str = "pyinstrument"):
        self.profiler_type = profiler_type
        self.profiler = None
        self.active = False
        
    def start(self):
        """Start profiling"""
        if self.profiler_type == "pyinstrument" and PYINSTRUMENT_AVAILABLE:
            self.profiler = pyinstrument.Profiler()
            self.profiler.start()
            self.active = True
        elif self.profiler_type == "line_profiler" and LINE_PROFILER_AVAILABLE:
            # line_profiler requires decoration, so we'll use a different approach
            self.profiler = line_profiler.LineProfiler()
            self.active = True
        else:
            print(f"WARNING: Profiler {self.profiler_type} not available, skipping profiling")
            self.active = False
    
    def stop(self) -> Optional[str]:
        """Stop profiling and return results"""
        if not self.active or self.profiler is None:
            return None
            
        if self.profiler_type == "pyinstrument":
            self.profiler.stop()
            # Get text output
            output = self.profiler.output_text(unicode=True, color=False)
            return output
        elif self.profiler_type == "line_profiler":
            # For line profiler, we'd need to add functions manually
            # This is more complex and would require code modifications
            return "Line profiler requires function decoration - use pyinstrument instead"
        
        return None
    
    def save_html_report(self, filename: str):
        """Save HTML report (pyinstrument only)"""
        if self.profiler_type == "pyinstrument" and self.profiler and self.active:
            try:
                html_output = self.profiler.output_html()
                with open(filename, 'w') as f:
                    f.write(html_output)
                return True
            except:
                return False
        return False


def timing_decorator(name: str, timings_dict: Dict[str, List[float]]):
    """Decorator to automatically time function calls"""
    def decorator(func):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            with DetailedTimer(name, timings_dict):
                return func(*args, **kwargs)
        return wrapper
    return decorator


def generate_test_tokens(num_tokens: int, vocab_size: int = 32000) -> torch.Tensor:
    """Generate deterministic test tokens for consistent cache hits"""
    # Use fixed seed for reproducible tokens
    torch.manual_seed(42)
    return torch.randint(0, vocab_size, (num_tokens,), dtype=torch.long)


def generate_kv_cache_paged_tensors(
    num_blocks: int,
    device: str = "cuda",
    block_size: int = 16,
    num_layers: int = 2,
    num_heads: int = 32,
    head_size: int = 128,
    dtype: torch.dtype = torch.bfloat16
) -> List[torch.Tensor]:
    """Generate paged KV cache tensors for testing
    
    For VLLMPagedMemGPUConnectorV2, each layer should have shape:
    [2, num_blocks, block_size, num_heads, head_size]
    where the first dimension is 2 for K and V tensors
    """
    # Use fixed seed for reproducible KV cache data
    torch.manual_seed(42)
    kv_cache = []
    
    # Create combined K+V tensor for each layer
    for layer_idx in range(num_layers):
        # Combined K+V tensor: [2, num_blocks, block_size, num_heads, head_size]
        layer_tensor = torch.randn(
            2, num_blocks, block_size, num_heads, head_size,
            dtype=dtype, device=device
        )
        kv_cache.append(layer_tensor)
    
    return kv_cache


def setup_lmcache_config(
    chunk_size: int,
    use_weka: bool = True,
    weka_path: str = "/mnt/weka/bench-cache",
    cufile_buffer_size: int = 16384,  # MB
    gds_io_threads: int = 4,  # Number of I/O threads
    local_cpu: bool = False,
    max_local_cpu_size: float = 5.0
) -> LMCacheEngineConfig:
    """Setup LMCache configuration"""
    if use_weka:
        return LMCacheEngineConfig.from_defaults(
            chunk_size=chunk_size,
            local_cpu=local_cpu,  # Disable CPU backend when using Weka
            max_local_cpu_size=max_local_cpu_size,
            weka_path=weka_path,
            cufile_buffer_size=cufile_buffer_size,
            gds_io_threads=gds_io_threads,  # Number of I/O threads
            remote_url=None,
            remote_serde=None,
            use_layerwise=False,
            save_decode_cache=False,
            enable_blending=False,
            enable_p2p=False,
            error_handling=False
        )
    else:
        return LMCacheEngineConfig.from_defaults(
            chunk_size=chunk_size,
            local_cpu=local_cpu,
            max_local_cpu_size=max_local_cpu_size,
            remote_url=None,
            remote_serde=None,
            use_layerwise=False,
            save_decode_cache=False,
            enable_blending=False,
            enable_p2p=False,
            error_handling=False
        )


def setup_lmcache_metadata(
    model_name: str = "benchmarkModel",
    world_size: int = 1,
    worker_id: int = 0,
    num_layers: int = 2,
    num_heads: int = 32,
    head_size: int = 128,
    chunk_size: int = 256
) -> LMCacheEngineMetadata:
    """Setup LMCache metadata"""
    # Include num_layers in model_name to ensure different layer configurations 
    # use different cache keys and don't share cache files
    # Note: Can't use '_' as storage backend has assertion against it
    model_name_with_layers = f"{model_name}-layers{num_layers}"
    
    return LMCacheEngineMetadata(
        model_name=model_name_with_layers,
        world_size=world_size,
        worker_id=worker_id,
        fmt="vllm",
        kv_dtype=torch.bfloat16,
        kv_shape=(num_layers, 2, chunk_size, num_heads, head_size),
        use_mla=False
    )


def clear_weka_cache_directory(weka_path: str, force: bool = False):
    """Clear the Weka cache directory to avoid corrupted file issues"""
    if force or input(f"Clear cache directory {weka_path}? (y/N): ").lower().startswith('y'):
        print(f"Clearing cache directory: {weka_path}")
        try:
            import subprocess
            import glob
            
            # Use glob to expand the wildcard pattern and remove each item
            items_to_remove = glob.glob(os.path.join(weka_path, "*"))
            if items_to_remove:
                subprocess.run(["rm", "-rf"] + items_to_remove, check=False, capture_output=True)
                print(f"Cache directory cleared successfully ({len(items_to_remove)} items removed)")
            else:
                print("Cache directory was already empty")
        except Exception as e:
            print(f"Warning: Failed to clear cache directory: {e}")


def enhanced_store_with_wait(engine, tokens, kv_cache, slot_mapping, timeout_seconds=30):
    """
    Enhanced store operation that waits for completion.
    This works around the fact that engine.store() doesn't return futures.
    """
    
    try:
        # Call the original store method
        engine.store(
            tokens,
            kvcaches=kv_cache,
            slot_mapping=slot_mapping
        )
        
        # Now wait for the storage to complete by monitoring put_tasks
        storage_backends = engine.storage_manager.storage_backends
        start_time = time.perf_counter()
        
        while time.perf_counter() - start_time < timeout_seconds:
            all_tasks_done = True
            
            for backend_name, backend in storage_backends.items():
                try:
                    # Check if this backend has pending put tasks
                    if hasattr(backend, 'put_tasks') and hasattr(backend, 'put_lock'):
                        with backend.put_lock:
                            if len(backend.put_tasks) > 0:
                                all_tasks_done = False
                                break
                except Exception:
                    # If we can't check the backend, continue
                    continue
            
            if all_tasks_done:
                return True
            
            # Brief sleep to avoid busy waiting
            time.sleep(0.05)
        
        # Timeout reached - return False to indicate we're not sure if complete
        return False
        
    except Exception as e:
        # Re-raise the original exception
        raise e








def benchmark_chunk_size(
    chunk_size: int,
    num_tokens: int,
    num_iterations: int = 3,
    enable_profiling: bool = False,
    profiler_type: str = "pyinstrument",
    detailed_timing: bool = True,
    use_weka: bool = True,
    weka_path: str = "/mnt/weka/bench-cache",
    cufile_buffer_size: int = 16384,
    gds_io_threads: int = 4,
    save_html_reports: bool = False,
    clear_cache: bool = False,
    device: str = "cuda",
    num_layers: int = 2,
    profiling_sleep_seconds: int = 0
) -> ProfilingResults:
    """Benchmark a specific chunk_size and num_tokens combination"""
    
    print(f"\n=== Benchmarking chunk_size={chunk_size}, tokens={num_tokens}, layers={num_layers} ===")
    
    # Calculate derived values
    num_chunks = (num_tokens + chunk_size - 1) // chunk_size  # Ceiling division
    num_blocks = (num_tokens + 15) // 16  # 16 tokens per block
    
    # Clear cache directory if requested
    if clear_cache and use_weka:
        clear_weka_cache_directory(weka_path, force=True)
    
    # Setup configuration and metadata
    config = setup_lmcache_config(
        chunk_size=chunk_size,
        use_weka=use_weka,
        weka_path=weka_path,
        cufile_buffer_size=cufile_buffer_size,
        gds_io_threads=gds_io_threads
    )
    metadata = setup_lmcache_metadata(chunk_size=chunk_size, num_layers=num_layers)
    
    # Create GPU connector
    gpu_connector = VLLMPagedMemGPUConnectorV2(
        hidden_dim_size=4096,  # Typical for 7B model
        num_layers=num_layers
    )
    
    # Create unique engine instance for this configuration
    engine_id = f"benchmark_chunk_{chunk_size}_tokens_{num_tokens}_layers_{num_layers}"
    
    profiling_output = None
    memory_errors = 0
    error_details = []
    
    try:
        # Create LMCache engine
        engine = LMCacheEngineBuilder.get_or_create(
            engine_id, config, metadata, gpu_connector
        )
        
        # Generate test data
        tokens = generate_test_tokens(num_tokens)
        kv_cache = generate_kv_cache_paged_tensors(num_blocks=num_blocks, device=device, num_layers=num_layers)
        # Use deterministic slot mapping for consistent results
        torch.manual_seed(42)
        slot_mapping = torch.randperm(num_blocks * 16, device=device)[:num_tokens]
        
        # Store data once at the beginning
        print(f"  Storing data once for cache...")
        torch.cuda.empty_cache()
        torch.cuda.synchronize()
        
        try:
            # Use enhanced store that waits for completion
            store_completed = enhanced_store_with_wait(
                engine, tokens, kv_cache, slot_mapping, timeout_seconds=30
            )
            torch.cuda.synchronize()  # Wait for GPU operations to complete
            
            if store_completed:
                print(f"  Store completed successfully (async operations finished)")
            else:
                print(f"  Store operation initiated (async operations may still be pending)")
                # Add additional wait time to be safe
                time.sleep(2.0)
                print(f"  Additional wait completed")
        except Exception as e:
            error_msg = f"Initial store failed: {str(e)}"
            print(f"    ERROR: {error_msg}")
            # Return failed result
            return ProfilingResults(
                chunk_size=chunk_size,
                num_tokens=num_tokens,
                num_chunks=num_chunks,
                retrieve_time=float('inf'),
                throughput_gbps=0,
                data_size_mb=0,
                detailed_timings={},
                profiling_output=None,
                memory_errors=1,
                error_details=[error_msg]
            )
        
        # Timing collectors for retrieve operations only
        detailed_timings = {}
        retrieve_times = []
        
        # Benchmark retrieve operations only
        for iteration in range(num_iterations):
            print(f"  Retrieve iteration {iteration + 1}/{num_iterations}")
            
            # Sleep before profiled iteration to allow external script coordination
            if enable_profiling and iteration == (num_iterations - 1) and profiling_sleep_seconds > 0:
                print(f"    Profiling iteration - sleeping {profiling_sleep_seconds} seconds for external script coordination...")
                time.sleep(profiling_sleep_seconds)
                print(f"    Starting profiled iteration...")
            
            # Prepare fresh cache tensors for retrieve operation
            retrieved_cache = generate_kv_cache_paged_tensors(num_blocks=num_blocks, device=device, num_layers=num_layers)
            torch.cuda.empty_cache()
            torch.cuda.synchronize()
            
            # Profile retrieve operation on last iteration (after warmup)
            retrieve_profiler = None
            if enable_profiling and iteration == (num_iterations - 1):
                retrieve_profiler = AdvancedProfiler(profiler_type)
                retrieve_profiler.start()
            
            iteration_retrieve_time = None
            try:
                if detailed_timing:
                    with DetailedTimer("retrieve_total", detailed_timings):
                        ret_mask = engine.retrieve(
                            tokens,
                            kvcaches=retrieved_cache,
                            slot_mapping=slot_mapping
                        )
                    # Get the time for this specific iteration
                    if detailed_timings.get("retrieve_total"):
                        iteration_retrieve_time = detailed_timings["retrieve_total"][-1]
                else:
                    start_time = time.perf_counter()
                    ret_mask = engine.retrieve(
                        tokens,
                        kvcaches=retrieved_cache,
                        slot_mapping=slot_mapping
                    )
                    torch.cuda.synchronize()
                    iteration_retrieve_time = time.perf_counter() - start_time
                    retrieve_times.append(iteration_retrieve_time)
                
                retrieved_tokens = torch.sum(ret_mask).item()
                print(f"    Retrieve time: {iteration_retrieve_time:.4f}s")
                print(f"    Retrieved: {retrieved_tokens}/{num_tokens} tokens")
                
            except Exception as e:
                memory_errors += 1
                error_msg = f"Retrieve error in iteration {iteration}: {str(e)}"
                error_details.append(error_msg)
                print(f"    ERROR: {error_msg}")
                # For 'NoneType' object has no attribute 'tensor' errors, 
                # this is likely due to storage backend returning None
                if "'NoneType' object has no attribute 'tensor'" in str(e):
                    error_details.append("This error suggests storage backend returned None (likely GDS file corruption)")
                continue
            
            if retrieve_profiler and retrieve_profiler.active:
                retrieve_output = retrieve_profiler.stop()
                if retrieve_output:
                    profile_filename = f"profile_retrieve_chunk_{chunk_size}_tokens_{num_tokens}_layers_{num_layers}.txt"
                    with open(profile_filename, 'w') as f:
                        f.write(f"Retrieve Operation Profile - Chunk Size: {chunk_size}, Tokens: {num_tokens}, Layers: {num_layers}\n")
                        f.write("="*80 + "\n")
                        f.write(retrieve_output)
                    print(f"  Retrieve profiling saved to {profile_filename}")
                    profiling_output = retrieve_output  # Store for results
                    
                    if save_html_reports:
                        html_filename = f"profile_retrieve_chunk_{chunk_size}_tokens_{num_tokens}_layers_{num_layers}.html"
                        if retrieve_profiler.save_html_report(html_filename):
                            print(f"  Retrieve HTML report saved to {html_filename}")
        
        # Print timing summary for this configuration
        print(f"  \n  === Timing Summary ===")
        
        # Collect all successful retrieve times for analysis
        successful_retrieve_times = []
        
        if detailed_timing:
            if "retrieve_total" in detailed_timings:
                successful_retrieve_times = detailed_timings["retrieve_total"]
        else:
            successful_retrieve_times = retrieve_times
        
        # Print individual retrieve times with outlier detection
        if successful_retrieve_times:
            retrieve_mean = np.mean(successful_retrieve_times)
            retrieve_std = np.std(successful_retrieve_times)
            retrieve_min = np.min(successful_retrieve_times)
            retrieve_max = np.max(successful_retrieve_times)
            retrieve_ratio = retrieve_max / retrieve_min if retrieve_min > 0 else float('inf')
            print(f"  Retrieve times: ", end="")
            for i, time_val in enumerate(successful_retrieve_times):
                # Mark outliers (more than 1.5 std deviations from mean)
                if abs(time_val - retrieve_mean) > 1.5 * retrieve_std and len(successful_retrieve_times) > 2:
                    print(f"{time_val:.4f}s* ", end="")  # * indicates outlier
                else:
                    print(f"{time_val:.4f}s ", end="")
            print(f"(avg: {retrieve_mean:.4f}s, std: {retrieve_std:.4f}s, max/min: {retrieve_ratio:.2f}x)")
        
        if successful_retrieve_times:
            print(f"  (* indicates outlier > 1.5 std dev from mean)")
        
        # Calculate results - handle case where no successful operations occurred          
        if detailed_timing and "retrieve_total" in detailed_timings:
            avg_retrieve_time = np.mean(detailed_timings["retrieve_total"])
        elif retrieve_times:
            avg_retrieve_time = np.mean(retrieve_times)
        else:
            avg_retrieve_time = float('inf')  # All retrieve operations failed
        
        # Calculate data size and throughput
        # Each token: num_layers * 2 (K,V) * 32 heads * 128 head_size * 2 bytes (bfloat16)
        data_size_bytes = num_tokens * num_layers * 2 * 32 * 128 * 2
        data_size_mb = data_size_bytes / (1024 * 1024)
        throughput_gbps = (data_size_mb / avg_retrieve_time / 1024) if avg_retrieve_time > 0 and avg_retrieve_time != float('inf') else 0
        
        # Aggregate detailed timings
        aggregated_timings = {}
        for key, times_list in detailed_timings.items():
            if times_list:  # Only process non-empty lists
                aggregated_timings[key] = {
                    'mean': np.mean(times_list),
                    'std': np.std(times_list),
                    'min': np.min(times_list),
                    'max': np.max(times_list)
                }
        
        results = ProfilingResults(
            chunk_size=chunk_size,
            num_tokens=num_tokens,
            num_chunks=num_chunks,
            retrieve_time=avg_retrieve_time,
            throughput_gbps=throughput_gbps,
            data_size_mb=data_size_mb,
            detailed_timings=aggregated_timings,
            profiling_output=profiling_output,
            memory_errors=memory_errors,
            error_details=error_details
        )
        
        # Print results with error information
        if memory_errors > 0:
            print(f"  ERRORS: Memory={memory_errors}")
            for error in error_details:
                print(f"    - {error}")
        
        print(f"  Results: Retrieve={avg_retrieve_time:.4f}s, "
              f"Throughput={throughput_gbps:.2f} GB/s")
        
        return results
    
    finally:
        # Cleanup
        try:
            LMCacheEngineBuilder.destroy(engine_id)
        except:
            pass


def run_comprehensive_benchmark(
    chunk_sizes: List[int],
    token_counts: List[int],
    num_iterations: int = 3,
    enable_profiling: bool = False,
    profiler_type: str = "pyinstrument",
    output_file: Optional[str] = None,
    use_weka: bool = True,
    weka_path: str = "/mnt/weka/bench-cache",
    cufile_buffer_size: int = 16384,
    gds_io_threads: int = 4,
    save_html_reports: bool = False,
    clear_cache: bool = False,
    device: str = "cuda",
    num_layers: int = 2,
    profiling_sleep_seconds: int = 0
) -> List[ProfilingResults]:
    """Run comprehensive benchmark across multiple configurations"""
    
    all_results = []
    
    print(f"Running advanced benchmark with:")
    print(f"  Chunk sizes: {chunk_sizes}")
    print(f"  Token counts: {token_counts}")
    print(f"  Iterations per config: {num_iterations}")
    print(f"  Profiling enabled: {enable_profiling}")
    print(f"  Profiler type: {profiler_type}")
    print(f"  Clear cache: {clear_cache}")
    
    for chunk_size in chunk_sizes:
        for num_tokens in token_counts:
            try:
                results = benchmark_chunk_size(
                    chunk_size=chunk_size,
                    num_tokens=num_tokens,
                    num_iterations=num_iterations,
                    enable_profiling=enable_profiling,
                    profiler_type=profiler_type,
                    use_weka=use_weka,
                    weka_path=weka_path,
                    cufile_buffer_size=cufile_buffer_size,
                    gds_io_threads=gds_io_threads,
                    save_html_reports=save_html_reports,
                    clear_cache=clear_cache and chunk_size == chunk_sizes[0] and num_tokens == token_counts[0],  # Only clear once
                    device=device,
                    num_layers=num_layers,
                    profiling_sleep_seconds=profiling_sleep_seconds
                )
                all_results.append(results)
                
                # Clean up GPU memory between tests
                torch.cuda.empty_cache()
                time.sleep(1)
                
            except Exception as e:
                print(f"ERROR in chunk_size={chunk_size}, tokens={num_tokens}: {e}")
                continue
    
    # Save results
    if output_file:
        # Convert results to JSON-serializable format
        json_results = []
        for result in all_results:
            json_result = {
                'chunk_size': result.chunk_size,
                'num_tokens': result.num_tokens,
                'num_chunks': result.num_chunks,
                'retrieve_time': result.retrieve_time if result.retrieve_time != float('inf') else None,
                'throughput_gbps': result.throughput_gbps,
                'data_size_mb': result.data_size_mb,
                'detailed_timings': result.detailed_timings,
                'has_profiling_data': result.profiling_output is not None,
                'memory_errors': result.memory_errors,
                'error_details': result.error_details
            }
            json_results.append(json_result)
        
        with open(output_file, 'w') as f:
            json.dump(json_results, f, indent=2)
        print(f"\nResults saved to {output_file}")
    
    # Print summary
    print(f"\n{'='*80}")
    print("ADVANCED BENCHMARK SUMMARY")
    print(f"{'='*80}")
    print(f"{'Chunk Size':<12} {'Tokens':<8} {'Chunks':<8} {'Retrieve(s)':<12} {'Throughput(GB/s)':<15} {'Errors':<10}")
    print(f"{'-'*75}")
    
    for result in all_results:
        retrieve_str = f"{result.retrieve_time:.4f}" if result.retrieve_time != float('inf') else "FAILED"
        error_str = f"M:{result.memory_errors}" if result.memory_errors > 0 else "-"
        
        print(f"{result.chunk_size:<12} {result.num_tokens:<8} {result.num_chunks:<8} "
              f"{retrieve_str:<12} {result.throughput_gbps:<15.2f} {error_str:<10}")
    
    return all_results


def main():
    parser = argparse.ArgumentParser(description="LMCache Advanced Profiling Benchmark")
    
    parser.add_argument(
        "--chunk-sizes",
        type=int,
        nargs='+',
        default=[64, 128, 256, 512, 1024],  # Reduced from [256, 512, 1024, 2048, 4096, 8192] for smaller files
        help="List of chunk sizes to test"
    )
    
    parser.add_argument(
        "--token-counts",
        type=int,
        nargs='+',
        default=[256, 512, 1024, 2048],  # Reduced from [2048, 4096, 8192, 16384] for smaller files
        help="List of token counts to test"
    )
    
    parser.add_argument(
        "--iterations",
        type=int,
        default=3,
        help="Number of iterations per configuration"
    )
    
    parser.add_argument(
        "--enable-profiling",
        action="store_true",
        help="Enable detailed profiling"
    )
    
    parser.add_argument(
        "--profiler-type",
        type=str,
        choices=["pyinstrument", "line_profiler"],
        default="pyinstrument",
        help="Type of profiler to use (default: pyinstrument)"
    )
    
    parser.add_argument(
        "--output",
        type=str,
        default="lmcache_advanced_profiling_results.json",
        help="Output file for results"
    )
    
    parser.add_argument(
        "--use-weka",
        action="store_true",
        default=True,
        help="Use Weka backend instead of local CPU (default: True)"
    )
    
    parser.add_argument(
        "--weka-path",
        type=str,
        default="/mnt/weka/bench-cache",
        help="Path to Weka mount point for cache storage"
    )
    
    parser.add_argument(
        "--cufile-buffer-size",
        type=int,
        default=16384,
        help="CuFile buffer size in MB for Weka backend"
    )
    
    parser.add_argument(
        "--gds-io-threads",
        type=int,
        default=4,
        help="Number of I/O threads for Weka GDS backend"
    )
    
    parser.add_argument(
        "--use-local-cpu",
        action="store_true",
        help="Use local CPU backend instead of Weka"
    )
    
    parser.add_argument(
        "--save-html-reports",
        action="store_true",
        help="Save HTML profiling reports (pyinstrument only)"
    )
    
    parser.add_argument(
        "--clear-cache",
        action="store_true",
        help="Clear the Weka cache directory before starting benchmark"
    )
    
    parser.add_argument(
        "--gpu-device",
        type=int,
        default=0,
        help="GPU device ID to use (default: 0)"
    )
    
    parser.add_argument(
        "--num-layers",
        type=int,
        default=2,
        help="Number of transformer layers to simulate (default: 2 for ~8MB files, use 32 for ~128MB files)"
    )
    
    parser.add_argument(
        "--profiling-sleep-seconds",
        type=int,
        default=0,
        help="Seconds to sleep before profiled iteration for external script coordination (default: 0, no sleep)"
    )
    
    args = parser.parse_args()
    
    # Validate profiler availability
    if args.enable_profiling:
        if args.profiler_type == "pyinstrument" and not PYINSTRUMENT_AVAILABLE:
            print("ERROR: pyinstrument not available. Install with: pip install pyinstrument")
            return 1
        elif args.profiler_type == "line_profiler" and not LINE_PROFILER_AVAILABLE:
            print("ERROR: line_profiler not available. Install with: pip install line_profiler")
            return 1

    if not LMCACHE_AVAILABLE:
        print("ERROR: lmcache and/or its deps are not available. Please run this from the AMG conda environment")
        return 1
    
    # Validate GPU availability
    if not torch.cuda.is_available():
        print("ERROR: CUDA is not available")
        return 1
    
    # Validate and set GPU device
    if args.gpu_device >= torch.cuda.device_count():
        print(f"ERROR: GPU device {args.gpu_device} not available. Available devices: 0-{torch.cuda.device_count()-1}")
        return 1
    
    torch.cuda.set_device(args.gpu_device)
    print(f"Using GPU {args.gpu_device}: {torch.cuda.get_device_name(args.gpu_device)}")
    print(f"GPU Memory: {torch.cuda.get_device_properties(args.gpu_device).total_memory / 1024**3:.1f} GB")
    
    # Calculate and display data size information
    bytes_per_token = args.num_layers * 2 * 32 * 128 * 2  # layers * (K,V) * heads * head_size * bytes_per_bfloat16
    kb_per_token = bytes_per_token / 1024
    print(f"Model configuration: {args.num_layers} layers, {kb_per_token:.1f} KB per token")
    
    # Determine backend configuration
    use_weka = not args.use_local_cpu  # Use Weka unless explicitly told to use local CPU
    
    backend_info = "Weka" if use_weka else "Local CPU"
    print(f"Using {backend_info} backend")
    if use_weka:
        print(f"  Weka path: {args.weka_path}")
        print(f"  CuFile buffer: {args.cufile_buffer_size} MB")
        print(f"  GDS I/O threads: {args.gds_io_threads}")
    
    if args.enable_profiling:
        print(f"Advanced profiling enabled with {args.profiler_type}")
        if args.save_html_reports and args.profiler_type == "pyinstrument":
            print("HTML reports will be generated")
    
    if args.clear_cache:
        print("Will clear cache directory before benchmark")
    
    # Run benchmark
    results = run_comprehensive_benchmark(
        chunk_sizes=args.chunk_sizes,
        token_counts=args.token_counts,
        num_iterations=args.iterations,
        enable_profiling=args.enable_profiling,
        profiler_type=args.profiler_type,
        output_file=args.output,
        use_weka=use_weka,
        weka_path=args.weka_path,
        cufile_buffer_size=args.cufile_buffer_size,
        gds_io_threads=args.gds_io_threads,
        save_html_reports=args.save_html_reports,
        clear_cache=args.clear_cache,
        device=f"cuda:{args.gpu_device}",
        num_layers=args.num_layers,
        profiling_sleep_seconds=args.profiling_sleep_seconds
    )
    
    # Print diagnostic information
    total_memory_errors = sum(r.memory_errors for r in results)
    
    print(f"\nAdvanced benchmark complete! Tested {len(results)} configurations.")
    if total_memory_errors > 0:
        print(f"Total errors: Memory={total_memory_errors}")
        print("\nSUGGESTED FIXES:")
        print("- Memory errors (NoneType tensor) are usually caused by storage failures")
        print("  * Check the error details above for specific failure causes")
        print("  * Try clearing the cache directory with --clear-cache")
        print("  * Try reducing the cufile-buffer-size (try 8192 or 4096)")
        print("  * Try reducing the number of gds-io-threads (try 2 or 1)")
        print("  * Try using --use-local-cpu instead of Weka backend")
    
    if args.enable_profiling:
        print("Check the generated profile_*.txt files for detailed CPU time analysis.")
        if args.save_html_reports:
            print("HTML reports are available for interactive viewing.")
    
    return 0


if __name__ == "__main__":
    exit(main()) 
