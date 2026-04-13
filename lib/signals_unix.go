//go:build !windows

package chef_load

import (
	"os"
	"os/signal"
	"syscall"
)

func notifySignals(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
}
