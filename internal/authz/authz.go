package authz

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/logic"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/policy"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type authz struct {
	sync.WaitGroup
	config struct {
		authorizationDisabled  bool
		publicSigningKeyMethod string
		publicSigningKey       string
		publicSigningKeyFile   string
		compilationInterval    time.Duration
		userId                 string
		policyCompileInputFile string
	}
	logger.Logger
	tracer.Tracer
	metrics.Metrics
	logic.Logic
	policy                 policy.Policy
	ctx                    context.Context
	ctxCancel              context.CancelFunc
	publicSigningKeyMethod jwt.SigningMethod
	publicKeyFunc          jwt.Keyfunc
	publicSigningKey       any
	opened                 atomic.Bool
	compileOutput          atomic.Pointer[data.PolicyCompileOutput]
	meter                  metrics.Meter
	counters               struct {
		sync.RWMutex
		data map[string]metrics.Int64Counter
	}
}

func New(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Authz
} {
	a := &authz{}
	for _, parameter := range parameters {
		switch v := parameter.(type) {
		case logic.Logic:
			a.Logic = v
		case policy.Policy:
			a.policy = v
		case tracer.Tracer:
			a.Tracer = v
		case metrics.Metrics:
			a.Metrics = v
		case logger.Logger:
			a.Logger = v
		}
	}
	return a
}

func (a *authz) createCounter(counterName string) (metrics.Int64UpDownCounter, error) {
	a.counters.Lock()
	defer a.counters.Unlock()
	counter, err := createCounter(a.meter, counterName)
	if err != nil {
		return nil, err
	}
	a.counters.data[counterName] = counter
	return counter, nil
}

func (a *authz) readCounter(counterName string) metrics.Int64Counter {
	a.counters.RLock()
	defer a.counters.RUnlock()
	return a.counters.data[counterName]
}

func (a *authz) evaluateBoolean(ctx context.Context, query string, input data.PolicyEvaluationInput) (bool, error) {
	var hasAccess bool

	if _, err := a.policy.EvaluatePrepared(ctx, query, input, &hasAccess); err != nil {
		return false, err
	}
	if hasAccess {
		counter := a.readCounter(counterNameAuthzAccessGranted)
		counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("query", query),
			attribute.String("user_id", input.TokenUserId),
		))
	} else {
		counter := a.readCounter(counterNameAuthzAccessDenied)
		counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("query", query),
			attribute.String("user_id", input.TokenUserId),
		))
	}
	return hasAccess, nil
}

func (a *authz) launchCompile() {
	started := make(chan struct{})
	a.Go(func() {
		const query string = "data.opa.compilation.resources"

		var previousChecksum string

		businessFx := func(ctx context.Context) error {
			compileInput := &data.PolicyCompileInput{}
			switch {
			default:
				compileInput = &data.PolicyCompileInput{
					Roles: map[string]data.PolicyRole{
						"employee_admin": {
							Name:    "employee_admin",
							UserIds: []string{a.config.userId},
							Permissions: []data.PolicyPermission{
								data.PolicyPermissionEmployeeCreate,
								data.PolicyPermissionEmployeeRead,
								data.PolicyPermissionEmployeeUpdate,
								data.PolicyPermissionEmployeeDelete,
							},
						},
					},
				}
			case a.config.policyCompileInputFile != "":
				bytes, err := os.ReadFile(a.config.policyCompileInputFile)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(bytes, compileInput); err != nil {
					return err
				}
			}
			currentChecksum, err := getChecksum(compileInput)
			if err != nil {
				return err
			}
			if previousChecksum == currentChecksum {
				return nil
			}
			//this ensures that we only get a trace when relevant work
			// is actually being done
			ctx, span := a.Start(ctx, "authz.launchCompile",
				trace.WithNewRoot())
			defer span.End()
			compileOutput := &data.PolicyCompileOutput{}
			if _, err := a.policy.CompilePrepared(ctx, query, compileInput,
				compileOutput); err != nil {
				return err
			}
			a.compileOutput.Store(compileOutput)
			previousChecksum = currentChecksum
			return nil
		}
		tCompile := time.NewTicker(a.config.compilationInterval)
		defer tCompile.Stop()
		close(started)
		if err := businessFx(a.ctx); err != nil {
			a.Error(a.ctx, "failed to execute authz compilation business logic", err)
		}
		for {
			select {
			case <-a.ctx.Done():
				return
			case <-tCompile.C:
				if err := businessFx(a.ctx); err != nil {
					a.Error(a.ctx, "failed to execute authz compilation business logic", err)
				}
			}
		}
	})
	<-started
}

func (a *authz) Configure(envs map[string]string) error {
	//set defaults
	a.config.compilationInterval = time.Second
	a.config.userId = "2ab78543-3c57-4a48-9a70-99d407aaf80e"

	//get configuration
	if s, ok := envs["AUTHZ_DISABLED"]; ok {
		a.config.authorizationDisabled, _ = strconv.ParseBool(s)
	}
	if publicSigningKeyMethod, ok := envs["AUTHZ_PUBLIC_SIGNING_METHOD"]; ok {
		a.config.publicSigningKeyMethod = publicSigningKeyMethod
	}
	if publicSigningKey, ok := envs["AUTHZ_PUBLIC_SIGNING_KEY"]; ok {
		a.config.publicSigningKey = publicSigningKey
	}
	if publicSigningKeyFile, ok := envs["AUTHZ_PUBLIC_SIGNING_KEY_FILE"]; ok {
		a.config.publicSigningKeyFile = publicSigningKeyFile
	}
	if userId, ok := envs["AUTHZ_USER_ID"]; ok {
		a.config.userId = userId
	}
	if policyCompileInputFile, ok := envs["AUTHZ_POLICY_COMPILE_INPUT_FILE"]; ok {
		a.config.policyCompileInputFile = policyCompileInputFile
	}
	return nil
}

func (a *authz) Open(ctx context.Context) error {
	if a.opened.Load() {
		return nil
	}
	if !a.config.authorizationDisabled {
		a.publicSigningKeyMethod = jwt.GetSigningMethod(a.config.publicSigningKeyMethod)
		if a.config.publicSigningKey == "" && a.config.publicSigningKeyFile == "" {
			return errors.Must(errors.New("no public signing key provided"))
		}
		switch a.publicSigningKeyMethod {
		default:
			return errors.Must(errors.New("unsupported public signing method"))
		case jwt.SigningMethodHS256:
			a.publicSigningKey = []byte(a.config.publicSigningKey)
		case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
			var rsaPublicKey *rsa.PublicKey
			var bytes []byte
			var err error

			switch {
			case a.config.publicSigningKey != "":
				rsaPublicKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(a.config.publicSigningKey))
			case a.config.publicSigningKeyFile != "":
				bytes, err = os.ReadFile(a.config.publicSigningKeyFile)
				if err != nil {
					return errors.Must(errors.New(fmt.Errorf("error while reading public key file (%s): %w",
						a.config.publicSigningKeyFile, err)))
				}
				rsaPublicKey, err = jwt.ParseRSAPublicKeyFromPEM(bytes)
			}
			if err != nil {
				return errors.Must(errors.New(fmt.Errorf("unable to parse rsa public key: %w", err)))
			}
			a.publicKeyFunc = func(t *jwt.Token) (any, error) {
				return rsaPublicKey, nil
			}
			a.publicSigningKey = rsaPublicKey
		}
		a.publicKeyFunc = func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return a.publicSigningKey, nil
		}
		a.compileOutput.Store(nil)
		a.ctx, a.ctxCancel = context.WithCancel(ctx)
		a.launchCompile()
	}
	meter := a.Meter(packageName)
	a.meter = meter
	a.counters.data = make(map[string]metrics.Int64Counter)
	for _, counterName := range counterNames {
		if _, err := a.createCounter(counterName); err != nil {
			return err
		}
	}
	a.opened.Store(true)
	return nil
}

func (a *authz) Close(ctx context.Context) {
	if !a.opened.Load() {
		return
	}
	a.ctxCancel()
	a.Wait()
	a.opened.Store(false)
}

func (a *authz) EmployeeCreate(ctx context.Context, authorization string,
	employeePartial data.EmployeePartial) (*data.Employee, error) {
	ctx, span := a.Start(ctx, "authz.EmployeeCreate")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			return nil, errors.Wrap(err, ErrJwt)
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			return nil, err
		}
		if !hasAccess {
			return nil, ErrUnauthorized(claims.UserId, "create", "employee", nil)
		}
	}
	return a.Logic.EmployeeCreate(ctx, employeePartial)
}

func (a *authz) EmployeeRead(ctx context.Context, authorization string, empNo int64) (*data.Employee, error) {
	ctx, span := a.Start(ctx, "authz.EmployeeRead")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			return nil, errors.Wrap(err, ErrJwt)
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			return nil, err
		}
		if !hasAccess {
			return nil, ErrUnauthorized(claims.UserId, "read",
				"employee", new(fmt.Sprint(empNo)))
		}
	}
	return a.Logic.EmployeeRead(ctx, empNo)
}

func (a *authz) EmployeesSearch(ctx context.Context, authorization string,
	search data.EmployeeSearch) ([]*data.Employee, error) {
	ctx, span := a.Start(ctx, "authz.EmployeesSearch")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			return nil, errors.Wrap(err, ErrJwt)
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			return nil, err
		}
		if !hasAccess {
			return nil, ErrUnauthorized(claims.UserId,
				"employee_search", "employee", nil)
		}
	}
	return a.Logic.EmployeesSearch(ctx, search)
}

func (a *authz) EmployeeUpdate(ctx context.Context, authorization string, empNo int64,
	employeePartial data.EmployeePartial) (*data.Employee, error) {
	ctx, span := a.Start(ctx, "authz.EmployeeUpdate")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			return nil, errors.Wrap(err, ErrJwt)
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			return nil, err
		}
		if !hasAccess {
			return nil, ErrUnauthorized(claims.UserId, "update",
				"employee", new(fmt.Sprint(empNo)))
		}
	}
	return a.Logic.EmployeeUpdate(ctx, empNo, employeePartial)
}

func (a *authz) EmployeeDelete(ctx context.Context, authorization string, empNo int64) error {
	ctx, span := a.Start(ctx, "authz.EmployeeDelete")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			return errors.Wrap(err, ErrJwt)
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			return err
		}
		if !hasAccess {
			return ErrUnauthorized(claims.UserId, "delete",
				"employee", new(fmt.Sprint(empNo)))
		}
	}
	return a.Logic.EmployeeDelete(ctx, empNo)
}
