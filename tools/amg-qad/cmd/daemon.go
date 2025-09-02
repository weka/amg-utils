package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/weka/amg-utils/tools/amg-qad/internal/config"
	"github.com/weka/amg-utils/tools/amg-qad/internal/scheduler"
	"github.com/weka/amg-utils/tools/amg-qad/internal/storage"
	"github.com/weka/amg-utils/tools/amg-qad/internal/web"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the QAD daemon",
	Long:  `Start the Quality Assurance Daemon that runs scheduled tests and provides a web dashboard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runOnce, _ := cmd.Flags().GetBool("run-once")
		return runDaemon(runOnce)
	},
}

func init() {
	// Set default configuration values
	viper.SetDefault("test_time", "23:59")
	viper.SetDefault("web_port", 9876)
	viper.SetDefault("results_path", "/mnt/weka/amg-qad/results/")

	// Add flags
	daemonCmd.Flags().BoolP("run-once", "o", false, "Run tests once and exit instead of running as daemon")
}

func runDaemon(runOnce bool) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	log.Printf("Starting AMG-QAD daemon version %s", version)
	log.Printf("Test schedule: %s", cfg.TestTime)
	log.Printf("Web dashboard: http://localhost:%d", cfg.WebPort)
	log.Printf("Results storage: %s", cfg.ResultsPath)

	// Initialize storage
	store, err := storage.New(cfg.ResultsPath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize scheduler
	sched := scheduler.New(cfg.TestTime, store)

	if runOnce {
		log.Println("Running in --run-once mode")
		// Run tests immediately and exit
		if err := sched.ExecuteTest(); err != nil {
			return fmt.Errorf("failed to execute test: %w", err)
		}
		log.Println("Test completed, exiting")
		return nil
	}

	// Initialize web server
	webServer := web.New(cfg.WebPort, store)

	// Start components
	if err := sched.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	if err := webServer.Start(); err != nil {
		return fmt.Errorf("failed to start web server: %w", err)
	}

	log.Println("AMG-QAD daemon started successfully")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down AMG-QAD daemon...")

	// Cleanup
	sched.Stop()
	webServer.Stop()

	log.Println("AMG-QAD daemon stopped")
	return nil
}

func init() {
	// Set default configuration values
	viper.SetDefault("test_time", "23:59")
	viper.SetDefault("web_port", 9876)
	viper.SetDefault("results_path", "/mnt/weka/amg-qad/results/")
}
