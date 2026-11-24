package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fnune/pocket-tunnel/internal/app"
	"github.com/fnune/pocket-tunnel/internal/minui"
	"github.com/fnune/pocket-tunnel/internal/netstate"
	"github.com/fnune/pocket-tunnel/internal/process"
	"github.com/fnune/pocket-tunnel/internal/store"
	"github.com/fnune/pocket-tunnel/internal/tunnel"
)

const iface = "wg0"

type systemClock struct{}

func (systemClock) Now() time.Time        { return time.Now() }
func (systemClock) Sleep(d time.Duration) { time.Sleep(d) }

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	flags := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	stateDir := flags.String("state-dir", "", "directory holding wg0.conf")
	binDir := flags.String("bin-dir", "", "directory holding minui-list and minui-presenter")
	autoSh := flags.String("auto-sh", "", "NextUI boot script the boot hook goes in")
	logFile := flags.String("log-file", "", "file the tunnel logs to")
	_ = flags.Parse(os.Args[2:])
	if *stateDir == "" {
		usage()
	}

	binary, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	upCommand := []string{binary, "up", "--state-dir", *stateDir}
	configs := store.New(*stateDir, iface, rand.Reader)
	pidFile := process.NewPIDFile(*stateDir)

	switch os.Args[1] {
	case "menu":
		menu := app.NewMenu(
			minui.New(*binDir),
			configs,
			process.NewTunnel(upCommand, *logFile, iface, pidFile),
			process.NewBootHook(*autoSh, upCommand, *logFile),
			netstate.New(iface),
			systemClock{},
		)
		if err := menu.Run(); err != nil {
			log.Fatal(err)
		}
	case "up":
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		up := app.NewUp(configs, pidFile, tunnel.NewRunner(iface, tunnel.ResolveUDP, tunnel.IP, tunnel.Wait))
		if err := up.Run(ctx); err != nil {
			log.Print(err)
			stop()
			os.Exit(1)
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: pocket-tunnel menu|up --state-dir DIR [--bin-dir DIR --auto-sh FILE --log-file FILE]")
	os.Exit(2)
}
