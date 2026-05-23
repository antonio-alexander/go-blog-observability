package internal

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/client"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
)

func generateToken(ctx context.Context, logger logger.Logger, envs map[string]string) (string, error) {
	var privateSigningKeyMethod, privateSigningKey, privateSigningKeyFile string

	userId := internal.GenerateId()
	if s, ok := envs["USER_ID"]; ok {
		userId = s
	}
	if s, ok := envs["AUTHZ_PRIVATE_SIGNING_KEY"]; ok {
		privateSigningKey = s
	}
	if s, ok := envs["AUTHZ_PRIVATE_SIGNING_METHOD"]; ok {
		privateSigningKeyMethod = s
	}
	if s, ok := envs["AUTHZ_PRIVATE_SIGNING_KEY_FILE"]; ok {
		privateSigningKeyFile = s
	}
	authzPrivateSigningMethod := jwt.GetSigningMethod(privateSigningKeyMethod)
	authzPrivateSigningKey, err := internal.GetPrivateSigningKey(privateSigningKeyMethod,
		privateSigningKey, privateSigningKeyFile)
	if err != nil {
		return "", err
	}
	logger.Info(ctx, "generating token", slog.String("user_id", userId))
	return jwt.NewWithClaims(authzPrivateSigningMethod, &data.AuthzClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute)},
			ID:        internal.GenerateId(),
		},
		UserId: userId,
	}).SignedString(authzPrivateSigningKey)
}

func Main(ctx context.Context, pwd string, args []string, envs map[string]string) error {
	var wg sync.WaitGroup
	var err error

	// create context we can cancel interactively
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	//create logger
	logger := logger.NewSlog()
	if err := logger.Configure(envs); err != nil {
		return err
	}
	if err := logger.Open(ctx); err != nil {
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
		client := client.NewClient(logger)
		if err := client.Configure(envs); err != nil {
			return err
		}
		if err := client.Open(ctx); err != nil {
			return err
		}
		defer client.Close(ctx)
		clients = append(clients, client)
	}

	// generate token
	token, err := generateToken(ctx, logger, envs)
	if err != nil {
		return err
	}

	// execute scenario
	scenario := envs["SCENARIO"]
	logger.Info(ctx, "executing scenario", slog.String("scenario", scenario))
	switch scenario {
	// TODO: largest input/output payloads
	case "percentile_latency":
		err = scenarioPercentileLatency(ctx, envs, logger, clients...)
	case "sustained_traffic":
		err = scenarioSustainedTraffic(ctx, envs, logger, clients...)
	case "stampeding_herd":
		err = scenarioStampedingHerd(ctx, envs, logger, token, clients...)
	case "cache_not_found":
		err = scenarioCacheNotFound(ctx, envs, logger, token, clients...)
	case "sleep_retry":
		err = scenarioSleepRetry(ctx, envs, logger, clients...)
	default:
		logger.Error(ctx, "unsupported scenario", slog.String("scenario", scenario))
	}
	cancel()
	wg.Wait()
	if err != nil {
		logger.Error(ctx, "scenario failed", slog.String("scenario", scenario), err)
	}
	return nil
}
