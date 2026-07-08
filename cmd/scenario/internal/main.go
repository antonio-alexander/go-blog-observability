package internal

import (
	"context"
	"log/slog"
	"strconv"
	"sync"

	"github.com/antonio-alexander/go-blog-observability/internal/client"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"
	"github.com/antonio-alexander/go-blog-observability/internal/scenario"
)

func Main(ctx context.Context, pwd string, args []string, envs map[string]string) error {
	var wg sync.WaitGroup
	var err error

	// create context we can cancel interactively
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	//create logger
	log := logger.NewSlog()
	switch envs["LOGGER_TYPE"] {
	case "open_telemetry":
		log = logger.NewOpenTelemetry()
	}
	if err := log.Configure(envs); err != nil {
		return err
	}
	if err := log.Open(ctx); err != nil {
		return err
	}

	//create traces
	traces := tracer.NewNull()
	switch envs["TRACER_TYPE"] {
	case "open_telemetry":
		traces = tracer.NewOpenTelemetry(log)
	}
	if err := traces.Configure(envs); err != nil {
		return err
	}
	if err := traces.Open(ctx); err != nil {
		return err
	}

	//get n clients configuration
	nClients := 1
	if s := envs["N_CLIENTS"]; s != "" {
		i, err := strconv.Atoi(s)
		if err == nil {
			nClients = i
		}
	}

	//create clients
	clients := make([]client.Client, 0, nClients)
	for range nClients {
		//create client
		client := client.NewClient(log, traces)
		if err := client.Configure(envs); err != nil {
			return err
		}
		if err := client.Open(ctx); err != nil {
			return err
		}
		defer client.Close(ctx)
		clients = append(clients, client)
	}

	// execute scenario
	scene := envs["SCENARIO"]
	log.Info(ctx, "executing scenario", slog.String("scenario", scene))
	switch scene {
	// TODO: largest input/output payloads
	case "percentile_latency":
		err = scenario.ScenarioPercentileLatency(ctx, envs, log, clients...)
	case "sustained_traffic":
		err = scenario.ScenarioSustainedTraffic(ctx, envs, log, clients...)
	case "stampeding_herd":
		err = scenario.ScenarioStampedingHerd(ctx, envs, log, clients...)
	case "cache_not_found":
		err = scenario.ScenarioCacheNotFound(ctx, envs, log, clients...)
	case "sleep_retry":
		err = scenario.ScenarioSleepRetry(ctx, envs, log, clients...)
	default:
		log.Error(ctx, "unsupported scenario", slog.String("scenario", scene))
	}
	cancel()
	wg.Wait()
	if err != nil {
		log.Error(ctx, "scenario failed", slog.String("scenario", scene), err)
	}
	return nil
}
