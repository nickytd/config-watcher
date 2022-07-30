package main

import (
	"config-watcher/metrics"
	"config-watcher/proc"
	"config-watcher/watcher"
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

var (
	config zap.Config

	// the target process command line that can be found under /proc/[pid]/cmdline
	cmdLine string

	// watched directory for configuration changes
	watchedDir string

	//log level flag
	debug bool

	//command line parameters handler
	rootCmd = &cobra.Command{
		Use:  "config-watcher",
		Long: "A simple tool noticing changes in files in a watched directory.",
	}

	//logger
	logger *zap.Logger

	// watched child process
	cmd *exec.Cmd

	// error
	err error
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
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, _ = config.Build()
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

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err = http.ListenAndServe(":8888", nil)
		if err != nil {
			log.Error(err.Error())
			os.Exit(-1)
		}
	}()

	hash := watcher.RunTotalHashCalc(ctx, watchedDir)
	currentHash := <-hash

	//Shall start the processes and maintain the PID
	cmd = startChildProcess(cmdLine)
	if err = cmd.Start(); err != nil {
		log.Error(err.Error())
		os.Exit(-1)
	}

	log.Info("process started",
		zap.Int("pid", cmd.Process.Pid),
		zap.String("state", cmd.ProcessState.String()))

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
				cmd, err = proc.RestartProcesses(ctx, cmd)
				if err = cmd.Start(); err != nil {
					log.Error(err.Error())
				}
				log.Info("process started",
					zap.Int("pid", cmd.Process.Pid))
				metrics.ProcssesRestarts()
			}
		}
	}

}

func startChildProcess(cmdLine string) *exec.Cmd {
	cmd := exec.Command(cmdLine, "-c", "/fluent-bit/etc/fluent-bit.conf")
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}
