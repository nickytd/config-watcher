package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	//utility logger supporting log levels
	logger = log.NewLogfmtLogger(os.Stdout)

	//enable debug logs
	debug bool

	// the target process command line that can be found under /proc/[pid]/cmdline
	cmdLine string

	// watched directory for configuration changes
	watchedDir string
)

// config-watcher calculates hashes of the files in the watchedDir and
// sends SIGTERM signal to a process when a change is detected
// Tools like fluent-bit doesn't implement a configuration reload hence a restart
// is needed to reload the configurations
// The config-watcher is deployed as a sidecar sharing the same process space
// with the target process in the pod

func main() {

	flag.BoolVar(&debug, "debug", false, "enable debug logs")
	flag.StringVar(&cmdLine, "cmdline", "/fluent-bit/bin/fluent-bit", "target process cmdline, example: /fluent-bit/bin/fluent-bit")
	flag.StringVar(&watchedDir, "watchedDir", "/fluent-bit/etc", "watched dir, example: /fluent-bit/etc ")
	flag.Parse()

	if debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	level.Debug(logger).Log("msg", "starting")
	watchedDir = strings.TrimSuffix(watchedDir, "/")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		sig := <-sigs
		level.Debug(logger).Log("receivedSignal", sig)
		done <- true
	}()

	go func() {
		hash := getSha256()
		for {
			level.Debug(logger).Log("hash", hash)
			if hash != getSha256() {
				pid := getTargetProcessPID()
				if pid != nil {
					level.Debug(logger).Log("targetProcessId", pid.Pid)
					if p, err := os.FindProcess(pid.Pid); err != nil {
						level.Error(logger).Log("error", err.Error())
					} else {
						if err := p.Signal(syscall.SIGTERM); err != nil {
							level.Error(logger).Log("error", err.Error())
						} else {
							hash = getSha256()
							level.Info(logger).Log("hash", hash)
						}
					}
				} else {
					level.Debug(logger).Log("msg", "process not found")
				}
			}
			time.Sleep(time.Second * 5)
		}
	}()

	<-done
	level.Info(logger).Log("msg", "exiting")

}

func getTargetProcessPID() *os.Process {

	dir, err := os.ReadDir("/proc")
	if err != nil {
		level.Error(logger).Log("error", err.Error())
		os.Exit(-1)
	}

	for _, f := range dir {
		if f.IsDir() {
			if pid, err := strconv.Atoi(f.Name()); err == nil {
				if content, err := ioutil.ReadFile("/proc/" + f.Name() + "/cmdline"); err != nil {
					level.Debug(logger).Log("error", err.Error())
					continue
				} else {
					if strings.Contains(string(content), cmdLine) {
						return &os.Process{
							Pid: pid,
						}
					} else {
						level.Debug(logger).Log("cmdline", content)
					}
				}
			}
		}
	}
	return nil
}

func getSha256() string {
	dir, err := os.ReadDir(watchedDir)
	if err != nil {
		level.Error(logger).Log("error", err.Error())
	}
	b := strings.Builder{}
	for _, f := range dir {
		fi, err := os.Stat(watchedDir + "/" + f.Name())
		if err != nil {
			level.Error(logger).Log("error", err.Error())
			continue
		}
		if fi.IsDir() {
			level.Debug(logger).Log("skipping folder", fi.Name())
			continue
		}

		h := sha256.New()
		t, err := os.Open(watchedDir + "/" + f.Name())
		if err != nil {
			level.Error(logger).Log("error", err.Error())
			continue
		}
		if _, err := io.Copy(h, t); err != nil {
			level.Error(logger).Log("error", err.Error())
			continue
		}
		t.Close()
		s := fmt.Sprintf("%x", h.Sum(nil))
		level.Debug(logger).Log(f.Name(), fmt.Sprintf("%x", h.Sum(nil)))
		b.Grow(len(s))
		b.WriteString(s)
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String())))
}
