package proc

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"os/exec"
	"syscall"
)

var log *zap.Logger

func RestartProcesses(ctx context.Context, cmd *exec.Cmd) (*exec.Cmd, error) {
	l := ctx.Value("logger")
	log = l.(*zap.Logger).Named("proc")

	if cmd == nil {
		log.Error("child process is nil")
		return nil, fmt.Errorf("invalid child processes")
	}

	pid := cmd.ProcessState.Pid()

	log.Info("current process",
		zap.Int("pid", pid))

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return nil, err
	}

	cmdLine := cmd.Path
	return exec.Command(cmdLine), nil

}
