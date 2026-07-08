package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/client"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/google/uuid"
)

func ScenarioStampedingHerd(ctx context.Context, envs map[string]string, logger logger.Logger,
	clients ...client.Client) error {
	const minClients int = 2

	var wg sync.WaitGroup

	readInterval := time.Second
	if s, ok := envs["SCENARIO_READ_INTERVAL"]; ok {
		i, _ := strconv.Atoi(s)
		readInterval = time.Duration(i) * time.Second
	}
	updateInterval := 2 * time.Second
	if s, ok := envs["SCENARIO_UPDATE_INTERVAL"]; ok {
		i, _ := strconv.Atoi(s)
		updateInterval = time.Duration(i) * time.Second
	}
	scenarioDuration := 10 * time.Second
	if s, ok := envs["SCENARIO_DURATION"]; ok {
		i, _ := strconv.Atoi(s)
		scenarioDuration = time.Duration(i) * time.Second
	}
	userId := uuid.NewString()
	if s, ok := envs["SCENARIO_USER_ID"]; ok {
		userId = s
	}
	if len(clients) < minClients {
		return errors.Must(errors.New("not enough clients provided"))
	}

	//generate context/correlation id
	correlationId := internal.GenerateId()
	ctx = pkgcontext.WithCorrelationId(ctx, correlationId)
	logger.Info(ctx, "generated correlation id",
		slog.String("scenario", "stampeding_herd"),
		slog.String("correlation_id", correlationId))

	//generate token
	token, err := clients[0].Authorize(ctx, userId)
	if err != nil {
		return err
	}
	logger.Info(ctx, "generated token",
		slog.String("token", token),
		slog.String("user_id", userId))

	// create employee using the first client
	birthDate := time.Now().Unix()
	firstName := internal.GenerateId()[:14]
	lastName := internal.GenerateId()[:16]
	gender, hireDate := "M", time.Now().Unix()
	employeeCreated, err := clients[0].EmployeeCreate(ctx, token,
		data.EmployeePartial{
			BirthDate: &birthDate,
			FirstName: &firstName,
			LastName:  &lastName,
			Gender:    &gender,
			HireDate:  &hireDate,
		})
	if err != nil {
		return err
	}
	empNo := employeeCreated.EmpNo
	defer func(empNo int64) {
		_ = clients[0].EmployeeDelete(ctx, token, empNo)
		logger.Info(ctx, "deleted employee",
			slog.String("employee_number", fmt.Sprint(empNo)))
	}(empNo)
	logger.Info(ctx, "created employee",
		slog.String("employee_number", fmt.Sprint(empNo)))

	//generate start/stop channels
	start, stop := make(chan struct{}), make(chan struct{})

	//create writer go routine
	wg.Add(1)
	go func(ctx context.Context, client client.Client) {
		defer wg.Done()

		ctx = pkgcontext.WithCorrelationId(ctx, correlationId)
		firstName := internal.GenerateId()[:14]
		lastName := internal.GenerateId()[:16]
		updateEmployeeFx := func(ctx context.Context) error {
			if _, err := client.EmployeeUpdate(ctx, token, empNo,
				data.EmployeePartial{
					FirstName: &firstName,
					LastName:  &lastName,
				}); err != nil {
				return err
			}
			return nil
		}
		tUpdate := time.NewTicker(updateInterval)
		defer tUpdate.Stop()
		<-start
		for {
			select {
			case <-stop:
				return
			case <-tUpdate.C:
				if err := updateEmployeeFx(ctx); err != nil {
					logger.Error(ctx, "unable to update employee", err)
				}
			}
		}
	}(ctx, clients[0])

	//create reader go routines
	for i := 1; i < len(clients); i++ {
		wg.Add(1)
		go func(ctx context.Context, clientNumber int, client client.Client) {
			defer wg.Done()

			readEmployeeFx := func(ctx context.Context) {
				if _, err := client.EmployeeRead(ctx, token, empNo); err != nil {
					logger.Error(ctx, "unable to read employee", err,
						slog.String("employee_number", fmt.Sprint(empNo)))
					return
				}
				logger.Info(ctx, "read employee",
					slog.String("employee_number", fmt.Sprint(empNo)))
			}
			tRead := time.NewTicker(readInterval)
			defer tRead.Stop()
			<-start
			for {
				select {
				case <-stop:
					return
				case <-tRead.C:
					readEmployeeFx(ctx)
				}
			}
		}(ctx, i, clients[i])
	}

	//start the go routines
	close(start)

	//allow go routines to run
	<-time.After(scenarioDuration)

	//stop go routines
	close(stop)
	wg.Wait()
	return nil
}

func ScenarioCacheNotFound(ctx context.Context, envs map[string]string, logger logger.Logger,
	clients ...client.Client) error {
	const minClients int = 1

	var employeeDeleted bool
	var wg sync.WaitGroup

	readInterval := time.Second
	if s, ok := envs["SCENARIO_READ_INTERVAL"]; ok {
		i, _ := strconv.Atoi(s)
		readInterval = time.Duration(i) * time.Second
	}
	scenarioDuration := 10 * time.Second
	if s, ok := envs["SCENARIO_DURATION"]; ok {
		i, _ := strconv.Atoi(s)
		scenarioDuration = time.Duration(i) * time.Second
	}
	userId := uuid.NewString()
	if s, ok := envs["SCENARIO_USER_ID"]; ok {
		userId = s
	}
	if len(clients) < minClients {
		return errors.Must(errors.New("not enough clients provided"))
	}

	//generate context/correlation id
	correlationId := internal.GenerateId()
	ctx = pkgcontext.WithCorrelationId(ctx, correlationId)
	logger.Info(ctx, "generated correlation id",
		slog.String("scenario", "cache_not_found"),
		slog.String("correlation_id", correlationId))

	//generate token
	token, err := clients[0].Authorize(ctx, userId)
	if err != nil {
		return err
	}
	logger.Info(ctx, "generated token",
		slog.String("token", token),
		slog.String("user_id", userId))

	// create employee
	birthDate := time.Now().Unix()
	firstName := internal.GenerateId()[:14]
	lastName := internal.GenerateId()[:16]
	gender, hireDate := "M", time.Now().Unix()
	employeeCreated, err := clients[0].EmployeeCreate(ctx, token,
		data.EmployeePartial{
			BirthDate: &birthDate,
			FirstName: &firstName,
			LastName:  &lastName,
			Gender:    &gender,
			HireDate:  &hireDate,
		})
	if err != nil {
		return err
	}
	empNo := employeeCreated.EmpNo
	defer func(empNo int64) {
		if !employeeDeleted {
			_ = clients[0].EmployeeDelete(ctx, token, empNo)
			logger.Info(ctx, "deleted employee: %d", empNo)
		}
	}(empNo)
	logger.Info(ctx, "created employee: %d", empNo)

	// delete created employee
	if err := clients[0].EmployeeDelete(ctx, token, empNo); err != nil {
		return err
	}
	employeeDeleted = true
	logger.Info(ctx, "deleted employee: %d", empNo)

	//generate start/stop channels
	start, stop := make(chan struct{}), make(chan struct{})

	//create go routines to read concurrently
	for i := range clients {
		wg.Add(1)
		go func(ctx context.Context, clientNumber int, client client.Client) {
			defer wg.Done()

			readEmployeeFx := func(ctx context.Context) error {
				if _, err := client.EmployeeRead(ctx, token, empNo); err != nil {
					return err
				}
				return nil
			}
			tRead := time.NewTicker(readInterval)
			defer tRead.Stop()
			<-start
			for {
				select {
				case <-stop:
					return
				case <-tRead.C:
					if err := readEmployeeFx(ctx); err != nil {
						switch {
						default:
							logger.Error(ctx, "error while reading employee", err)
						case errors.Is(err, errors.ErrNotCached):
							logger.Info(ctx, "employee not cached")
						}
					}
				}
			}
		}(ctx, i, clients[i])
	}

	//clear cache counters and start the go routines
	if err := clients[0].CacheClear(ctx); err != nil {
		return err
	}

	//start the scenarios
	close(start)

	//allow go routines to run
	<-time.After(scenarioDuration)

	//stop go routines
	close(stop)
	wg.Wait()

	return nil
}

func ScenarioSleepRetry(ctx context.Context, envs map[string]string, logger logger.Logger,
	clients ...client.Client) error {
	const minClients int = 1

	var wg sync.WaitGroup

	readInterval := time.Second
	if s, ok := envs["SCENARIO_READ_INTERVAL"]; ok {
		i, _ := strconv.Atoi(s)
		readInterval = time.Duration(i) * time.Second
	}
	scenarioDuration := 10 * time.Second
	if s, ok := envs["SCENARIO_DURATION"]; ok {
		i, _ := strconv.Atoi(s)
		scenarioDuration = time.Duration(i) * time.Second
	}
	scenarioSleepDuration := int64(1)
	if s, ok := envs["SCENARIO_SLEEP_DURATION"]; ok {
		scenarioSleepDuration, _ = strconv.ParseInt(s, 10, 64)
	}
	scenarioSleepId := ""
	if s, ok := envs["SCENARIO_GENERATE_SLEEP_ID"]; ok {
		b, err := strconv.ParseBool(s)
		if err == nil && b {
			scenarioSleepId = internal.GenerateId()
			logger.Info(ctx, "generated sleep id",
				slog.String("scenario", "sleep_retry"),
				slog.String("sleep_id", scenarioSleepId))
		}
	}
	if len(clients) < minClients {
		return errors.Must(errors.New("not enough clients provided"))
	}

	//generate context/correlation id
	correlationId := internal.GenerateId()
	ctx = pkgcontext.WithCorrelationId(ctx, correlationId)
	logger.Info(ctx, "generated correlation id",
		slog.String("scenario", "sleep_retry"),
		slog.String("correlation_id", correlationId))

	//generate start/stop channels
	start, stop := make(chan struct{}), make(chan struct{})

	//create go routines to sleep concurrently
	for i := range clients {
		wg.Add(1)
		go func(ctx context.Context, clientNumber int, client client.Client) {
			defer wg.Done()

			sleepFx := func(ctx context.Context) error {
				if _, err := client.Sleep(ctx, data.Sleep{
					Id:       scenarioSleepId,
					Duration: scenarioSleepDuration,
				}); err != nil {
					return err
				}
				return nil
			}
			tRead := time.NewTicker(readInterval)
			defer tRead.Stop()
			<-start
			for {
				select {
				case <-stop:
					return
				case <-tRead.C:
					if err := sleepFx(ctx); err != nil {
						logger.Error(ctx, "error while sleeping", err)
					}
				}
			}
		}(ctx, i, clients[i])
	}

	//start the scenarios
	close(start)

	//allow go routines to run
	<-time.After(scenarioDuration)

	//stop go routines
	close(stop)
	wg.Wait()

	return nil
}

func ScenarioSustainedTraffic(ctx context.Context, envs map[string]string, logger logger.Logger,
	clients ...client.Client) error {
	const minClients int = 1

	var wg sync.WaitGroup

	sleepInterval := time.Second
	if s, ok := envs["SCENARIO_SLEEP_INTERVAL"]; ok {
		i, err := strconv.Atoi(s)
		if err == nil {
			sleepInterval = time.Duration(i) * time.Second
		}
	}
	scenarioDuration := 10 * time.Second
	if s, ok := envs["SCENARIO_DURATION"]; ok {
		i, err := strconv.Atoi(s)
		if err == nil {
			scenarioDuration = time.Duration(i) * time.Second
		}
	}
	scenarioSleepDuration := int64(1)
	if s, ok := envs["SCENARIO_SLEEP_DURATION"]; ok {
		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			scenarioSleepDuration = i
		}
	}
	scenarioSleepId := ""
	if s, ok := envs["SCENARIO_GENERATE_SLEEP_ID"]; ok {
		b, err := strconv.ParseBool(s)
		if err == nil && b {
			scenarioSleepId = internal.GenerateId()
			logger.Info(ctx, "generated sleep id",
				slog.String("scenario", "sustained_traffic"),
				slog.String("sleep_id", scenarioSleepId))

		}
	}
	scenarioClearCache := false
	if s, ok := envs["SCENARIO_CLEAR_CACHE"]; ok {
		b, err := strconv.ParseBool(s)
		if err == nil {
			scenarioClearCache = b
		}
	}
	if len(clients) < minClients {
		return errors.Must(errors.New("not enough clients provided"))
	}

	//generate context/correlation id
	correlationId := internal.GenerateId()
	ctx = pkgcontext.WithCorrelationId(ctx, correlationId)
	logger.Info(ctx, "generated correlation id",
		slog.String("scenario", "sustained_traffic"),
		slog.String("correlation_id", correlationId))

	//clear cache
	if scenarioClearCache {
		if err := clients[0].CacheClear(ctx); err != nil {
			return err
		}
	}

	//cache the sleep so we can always engage the logic
	// in the same way
	if _, err := clients[0].Sleep(ctx, data.Sleep{
		Id:       scenarioSleepId,
		Duration: scenarioSleepDuration,
	}); err != nil {
		return err
	}

	//generate start/stop channels
	start, stop := make(chan struct{}), make(chan struct{})

	//create go routines to sleep concurrently
	for i := range clients {
		wg.Add(1)
		go func(ctx context.Context, clientNumber int, client client.Client) {
			defer wg.Done()

			sleepFx := func(ctx context.Context) error {
				if _, err := client.Sleep(ctx, data.Sleep{
					Id:       scenarioSleepId,
					Duration: scenarioSleepDuration,
				}); err != nil {
					return err
				}
				return nil
			}
			tRead := time.NewTicker(sleepInterval)
			defer tRead.Stop()
			<-start
			for {
				select {
				case <-stop:
					return
				case <-tRead.C:
					if err := sleepFx(ctx); err != nil {
						logger.Error(ctx, "error while sleeping", err)
					}
				}
			}
		}(ctx, i, clients[i])
	}

	//start the scenarios
	close(start)

	//allow go routines to run
	<-time.After(scenarioDuration)

	//stop go routines
	close(stop)
	wg.Wait()

	return nil
}

func ScenarioPercentileLatency(ctx context.Context, envs map[string]string, logger logger.Logger,
	clients ...client.Client) error {
	const minClients int = 10

	var wg sync.WaitGroup

	sleepInterval := time.Second
	if s, ok := envs["SCENARIO_SLEEP_INTERVAL"]; ok {
		i, err := strconv.Atoi(s)
		if err == nil {
			sleepInterval = time.Duration(i) * time.Second
		}
	}
	scenarioDuration := 10 * time.Second
	if s, ok := envs["SCENARIO_DURATION"]; ok {
		i, err := strconv.Atoi(s)
		if err == nil {
			scenarioDuration = time.Duration(i) * time.Second
		}
	}
	scenarioSleepId := ""
	if s, ok := envs["SCENARIO_GENERATE_SLEEP_ID"]; ok {
		b, err := strconv.ParseBool(s)
		if err == nil && b {
			scenarioSleepId = internal.GenerateId()
			logger.Info(ctx, "generated sleep id",
				slog.String("scenario", "percentile_latency"),
				slog.String("sleep_id", scenarioSleepId))
		}
	}
	scenarioPercentile := int(99)
	if s, ok := envs["SCENARIO_PERCENTILE"]; ok {
		i, err := strconv.Atoi(s)
		if err == nil {
			scenarioPercentile = i
		}
	}
	minSleep := int64(1) //1s
	if s, ok := envs["SCENARIO_MIN_SLEEP"]; ok {
		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			minSleep = i
		}
	}
	maxSleep := int64(5) //5s
	if s, ok := envs["SCENARIO_MAX_SLEEP"]; ok {
		i, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			maxSleep = i
		}
	}
	if len(clients) < minClients {
		return errors.Must(errors.New("not enough clients provided"))
	}

	//generate context/correlation id
	correlationId := internal.GenerateId()
	ctx = pkgcontext.WithCorrelationId(ctx, correlationId)
	logger.Info(ctx, "generated correlation id",
		slog.String("scenario", "percentile_latency"),
		slog.String("correlation_id", correlationId))

	//determine percentiles
	sleepDurations := make([]int64, 0, len(clients))
	switch {
	default:
		return errors.Must(errors.New("unsupported scenario percentile"))
	case scenarioPercentile >= 90:
		// how to get n == 9
		n := 9
		if len(clients)%n == 0 {
			n--
		}
		for range n {
			sleepDurations = append(sleepDurations, minSleep)
		}
		for range len(clients) - n {
			sleepDurations = append(sleepDurations, maxSleep)
		}
	case scenarioPercentile == 50: //simulate conditions p50
		n := 5
		if len(clients)%n == 0 {
			n--
		}
		for range n {
			sleepDurations = append(sleepDurations, minSleep)
		}
		for range len(clients) - n {
			sleepDurations = append(sleepDurations, maxSleep)
		}
	}

	//no real need to interact with the cache

	//generate start/stop channels
	start, stop := make(chan struct{}), make(chan struct{})

	//start go routines
	for i := range clients {
		wg.Add(1)
		go func(ctx context.Context, clientNumber int, client client.Client, sleepDuration int64) {
			defer wg.Done()

			var nRequests, nRequestsFailed int

			sleepFx := func(ctx context.Context) error {
				if _, err := client.Sleep(ctx, data.Sleep{
					Id:       scenarioSleepId,
					Duration: sleepDuration,
				}); err != nil {
					return err
				}
				return nil
			}
			tRead := time.NewTicker(sleepInterval)
			defer tRead.Stop()
			<-start
			for {
				select {
				case <-stop:
					logger.Info(ctx, "execution completed",
						slog.String("scenario", "percentile_latency"),
						slog.Int("client_number", clientNumber),
						slog.Int("number_of_requests", nRequests),
						slog.Int("number_of_failed_requests", nRequestsFailed),
						slog.Int64("sleep_duration", sleepDuration))
					return
				case <-tRead.C:
					if err := sleepFx(ctx); err != nil {
						logger.Error(ctx, "error while sleeping", err)
						nRequestsFailed++
						continue
					}
					nRequests++
				}
			}
		}(ctx, i, clients[i], sleepDurations[i])
	}

	//start the scenarios
	close(start)

	//allow go routines to run
	<-time.After(scenarioDuration)

	//stop go routines
	close(stop)
	wg.Wait()

	return nil
}
