package cmd

// Default values for vLLM and AMG configuration
// These constants are used across docker launch and host launch commands
// to ensure consistency and avoid magic numbers in the codebase.
const (
	// vLLM Configuration Defaults
	DefaultGPUMemUtil       = 0.8
	DefaultMaxSequences     = 256
	DefaultMaxModelLen      = 16384
	DefaultMaxBatchedTokens = 16384
	DefaultPort             = 8000

	// Filesystem Defaults
	DefaultWekaMount = "/mnt/weka"
	DefaultHFHome    = "/mnt/weka/hf_cache"

	// LMCache Configuration Defaults
	DefaultLMCachePath            = "/mnt/weka/cache"
	DefaultLMCacheChunkSize       = 256
	DefaultLMCacheGDSThreads      = 32
	DefaultLMCacheCuFileBuffer    = "8192"
	DefaultLMCacheSaveDecodeCache = true

	// Prometheus Configuration Defaults
	DefaultPrometheusMultiprocDir = "/tmp/lmcache_prometheus"
)
