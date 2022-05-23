package proc

import (
	"context"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var log *zap.Logger

func getTargetProcessPID(cmdLine string) *os.Process {

	dir, err := os.ReadDir("/proc")
	if err != nil {
		log.Error(
			"error reading proc",
			zap.Error(err),
		)
	}

	for _, f := range dir {
		if f.IsDir() {
			if pid, err := strconv.Atoi(f.Name()); err == nil {
				if content, err := ioutil.ReadFile("/proc/" + f.Name() + "/cmdline"); err != nil {
					log.Error(
						"error reading proc",
						zap.Error(err),
					)
					continue
				} else {
					if strings.Contains(string(content), cmdLine) {
						return &os.Process{
							Pid: pid,
						}
					} else {
						log.Debug(
							"reading proc.",
							zap.String("cmdline", cmdLine),
						)
					}
				}
			}
		}
	}
	return nil
}

func TerminateProcess(ctx context.Context, cmdLine string) {
	if l := ctx.Value("logger"); l != nil {
		log = l.(*zap.Logger).Named("proc")
	} else {
		log = zap.NewNop()
	}
	pid := getTargetProcessPID(cmdLine)

	if pid != nil {
		log.Debug(
			"found process",
			zap.Int("pid", pid.Pid),
		)
		if p, err := os.FindProcess(pid.Pid); err != nil {
			log.Error(
				"cannot find pricess",
				zap.Int("pid", pid.Pid),
			)
		} else {
			if err := p.Signal(syscall.SIGTERM); err != nil {
				log.Error(
					"error sedning TERM signal",
					zap.Int("pid", pid.Pid),
				)
			}
		}
	} else {
		log.Debug(
			"process not found",
			zap.String("cmdline", cmdLine),
		)
	}
}
