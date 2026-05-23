package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/antonio-alexander/go-blog-observability/cmd/service/internal"
)

func main() {
	pwd, _ := os.Getwd()
	args := os.Args[1:]
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := internal.Main(ctx, pwd, args, envs); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}
