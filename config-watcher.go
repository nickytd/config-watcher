package main

import (
	"context"
	"fmt"
	"github.com/nickytd/config-watcher/metrics"
	"github.com/nickytd/config-watcher/proc"
	"github.com/nickytd/config-watcher/watcher"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (

	// the target process command line that can be found under /proc/[pid]/cmdline
	cmdLine string

	// watched directory for configuration changes
	watchedDir string

	//log level flag
	debug bool

	//command line parametes handler
	rootCmd = &cobra.Command{
		Use:  "config-watcher",
		Long: "A simple tool noticing changes in files in a watched directory.",
	}

	//logger
	logger *zap.Logger
)

// config-watcher calculates hashes of the files in the watchedDir and
// sends SIGTERM signal to a process when a change is detected
// Tools like fluent-bit doesn't implement a configuration reload hence a restart
// is needed to reload the configurations
// The config-watcher is deployed as a sidecar sharing the same process space
// with the target process in the pod

func main() {
	rootCmd.Flags().StringVarP(
		&cmdLine,
		"cmdline",
		"c",
		"/fluent-bit/bin/fluent-bit",
		"target process cmdline, example: /fluent-bit/bin/fluent-bit",
	)

	rootCmd.Flags().StringVarP(
		&watchedDir,
		"watchedDir",
		"w",
		"/fluent-bit/etc",
		"watched dir, example: /fluent-bit/etc ",
	)

	rootCmd.Flags().BoolVarP(
		&debug,
		"debug",
		"d",
		false,
		"enables debug log level",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if debug {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}

	log := logger.Named("main")
	log.Info("starting")

	watchedDir = strings.TrimSuffix(watchedDir, "/")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		sig := <-sigs
		log.Info(
			"signal received",
			zap.String("signal", sig.String()),
		)
		done <- true
	}()

	c, cancel := context.WithCancel(context.Background())
	ctx := context.WithValue(c, "logger", logger)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8888", nil)

	hash := watcher.RunTotalHashCalc(ctx, watchedDir)
	currentHash := <-hash
	for {
		select {
		case <-done:
			cancel()
			log.Info("exiting")
			os.Exit(0)
		case h := <-hash:
			if currentHash != h {
				log.Info(
					"total hash changed",
					zap.String("old hash", currentHash),
					zap.String("new hash", h),
				)
				currentHash = h
				metrics.IncreaseTotalHashUpdates()
				metrics.ResetFileHash()
				go proc.TerminateProcess(ctx, cmdLine)
			}
		}
	}

}
