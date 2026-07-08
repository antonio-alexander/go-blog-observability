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
	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type authz struct {
	sync.WaitGroup
	logger.Logger
	tracer.Tracer
	metrics.Metrics
	*metrics.Int64CounterHelper
	logic.Logic
	config struct {
		authorizationDisabled   bool
		publicSigningKeyMethod  string
		publicSigningKey        string
		publicSigningKeyFile    string
		privateSigningKeyMethod string
		privateSigningKey       string
		privateSigningKeyFile   string
		compilationInterval     time.Duration
		userId                  string
		policyCompileInputFile  string
	}
	policy                  policy.Policy
	ctx                     context.Context
	ctxCancel               context.CancelFunc
	publicSigningKeyMethod  jwt.SigningMethod
	publicKeyFunc           jwt.Keyfunc
	publicSigningKey        any
	privateSigningKeyMethod jwt.SigningMethod
	privateSigningKey       any
	opened                  atomic.Bool
	compileOutput           atomic.Pointer[data.PolicyCompileOutput]
}

func New(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Authz
} {
	a := &authz{
		Int64CounterHelper: metrics.NewInt64CounterHelper(),
	}
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

func (a *authz) evaluateBoolean(ctx context.Context, query string, input data.PolicyEvaluationInput) (bool, error) {
	var hasAccess bool

	if _, err := a.policy.EvaluatePrepared(ctx, query, input, &hasAccess); err != nil {
		return false, err
	}

	metricOption := metric.WithAttributes(
		attribute.String("query", query),
		attribute.String("user_id", input.TokenUserId),
	)
	if hasAccess {
		a.Int64CounterHelper.Add(ctx, counterNameAuthzAccessGranted,
			1, metricOption)
		return hasAccess, nil
	}
	a.Int64CounterHelper.Add(ctx, counterNameAuthzAccessDenied,
		1, metricOption)
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
			ctx, span := a.Start(ctx, "authz.launch_compile",
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
	if privateSigningKeyMethod, ok := envs["AUTHZ_PRIVATE_SIGNING_METHOD"]; ok {
		a.config.privateSigningKeyMethod = privateSigningKeyMethod
	}
	if privateSigningKey, ok := envs["AUTHZ_PRIVATE_SIGNING_KEY"]; ok {
		a.config.privateSigningKey = privateSigningKey
	}
	if privateSigningKeyFile, ok := envs["AUTHZ_PRIVATE_SIGNING_KEY_FILE"]; ok {
		a.config.privateSigningKeyFile = privateSigningKeyFile
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
		a.privateSigningKeyMethod = jwt.GetSigningMethod(a.config.privateSigningKeyMethod)
		if a.config.privateSigningKey == "" && a.config.privateSigningKeyFile == "" {
			return errors.Must(errors.New("no private signing key provided"))
		}
		switch a.privateSigningKeyMethod {
		default:
			return errors.Must(errors.New("unsupported private signing method"))
		case jwt.SigningMethodHS256:
			a.privateSigningKey = []byte(a.config.privateSigningKey)
		case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
			var rsaPrivateKey *rsa.PrivateKey
			var bytes []byte
			var err error

			switch {
			case a.config.privateSigningKey != "":
				rsaPrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(a.config.privateSigningKey))
			case a.config.privateSigningKeyFile != "":
				bytes, err = os.ReadFile(a.config.privateSigningKeyFile)
				if err != nil {
					return errors.Must(errors.New(fmt.Errorf("error while reading private key file (%s): %w",
						a.config.privateSigningKeyFile, err)))
				}
				rsaPrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM(bytes)
			}
			if err != nil {
				return errors.Must(errors.New(fmt.Errorf("unable to parse rsa private key: %w", err)))
			}
			a.privateSigningKey = rsaPrivateKey
		}
		a.compileOutput.Store(nil)
		a.ctx, a.ctxCancel = context.WithCancel(ctx)
		a.launchCompile()
	}
	meter := a.Meter(packageName)
	if err := createCounters(meter, a.Int64CounterHelper); err != nil {
		return err
	}
	a.opened.Store(true)
	return nil
}

func (a *authz) Close(ctx context.Context) {
	if !a.opened.Load() {
		return
	}
	a.opened.Store(false)
	a.ctxCancel()
	a.Wait()
	a.Int64CounterHelper.Close(ctx)
}

func (a *authz) Authorize(ctx context.Context, userId string) (string, error) {
	if !a.opened.Load() {
		return "", ErrAuthzNotOpened
	}
	ctx, span := a.Start(ctx, "authz.employee_create")
	defer span.End()
	if a.config.authorizationDisabled {
		err := errors.Must(errors.New("authorization disabled"))
		span.RecordError(err)
		return "", err
	}
	t := time.Now()
	claims := &data.AuthzClaims{
		Issuer:    "go-blog-observability",
		Audience:  jwt.ClaimStrings{},
		ExpiresAt: &jwt.NumericDate{Time: t.Add(time.Hour)},
		NotBefore: &jwt.NumericDate{Time: t},
		IssuedAt:  &jwt.NumericDate{Time: t},
		ID:        uuid.NewString(),
		UserId:    userId,
	}
	authorization, err := jwt.NewWithClaims(a.publicSigningKeyMethod, claims).SignedString(a.privateSigningKey)
	if err != nil {
		span.RecordError(err)
		return "", err
	}
	return authorization, nil
}

func (a *authz) EmployeeCreate(ctx context.Context, authorization string,
	employeePartial data.EmployeePartial) (*data.Employee, error) {
	if !a.opened.Load() {
		return nil, ErrAuthzNotOpened
	}
	ctx, span := a.Start(ctx, "authz.employee_create")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			err := errors.Wrap(err, ErrJwt)
			span.RecordError(err)
			return nil, err
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		if !hasAccess {
			err := ErrUnauthorized(claims.UserId, "create", "employee", nil)
			span.RecordError(err)
			return nil, err
		}
	}
	employee, err := a.Logic.EmployeeCreate(ctx, employeePartial)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return employee, nil
}

func (a *authz) EmployeeRead(ctx context.Context, authorization string, empNo int64) (*data.Employee, error) {
	if !a.opened.Load() {
		return nil, ErrAuthzNotOpened
	}
	ctx, span := a.Start(ctx, "authz.employee_read")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			err := errors.Wrap(err, ErrJwt)
			span.RecordError(err)
			return nil, err
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		if !hasAccess {
			err := ErrUnauthorized(claims.UserId, "read",
				"employee", new(fmt.Sprint(empNo)))
			span.RecordError(err)
			return nil, err
		}
	}
	employee, err := a.Logic.EmployeeRead(ctx, empNo)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return employee, nil
}

func (a *authz) EmployeesSearch(ctx context.Context, authorization string,
	search data.EmployeeSearch) ([]*data.Employee, error) {
	if !a.opened.Load() {
		return nil, ErrAuthzNotOpened
	}
	ctx, span := a.Start(ctx, "authz.employee_search")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			err := errors.Wrap(err, ErrJwt)
			span.RecordError(err)
			return nil, err
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		if !hasAccess {
			err := ErrUnauthorized(claims.UserId,
				"employee_search", "employee", nil)
			span.RecordError(err)
			return nil, err
		}
	}
	employees, err := a.Logic.EmployeesSearch(ctx, search)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return employees, nil
}

func (a *authz) EmployeeUpdate(ctx context.Context, authorization string, empNo int64,
	employeePartial data.EmployeePartial) (*data.Employee, error) {
	if !a.opened.Load() {
		return nil, ErrAuthzNotOpened
	}
	ctx, span := a.Start(ctx, "authz.employee_update")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			err := errors.Wrap(err, ErrJwt)
			span.RecordError(err)
			return nil, err
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		if !hasAccess {
			err := ErrUnauthorized(claims.UserId, "update",
				"employee", new(fmt.Sprint(empNo)))
			span.RecordError(err)
			return nil, err
		}
	}
	employee, err := a.Logic.EmployeeUpdate(ctx, empNo, employeePartial)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return employee, nil
}

func (a *authz) EmployeeDelete(ctx context.Context, authorization string, empNo int64) error {
	if !a.opened.Load() {
		return ErrAuthzNotOpened
	}
	ctx, span := a.Start(ctx, "authz.employee_delete")
	defer span.End()
	if !a.config.authorizationDisabled {
		claims := &data.AuthzClaims{}
		if _, err := jwt.ParseWithClaims(authorization, claims,
			a.publicKeyFunc); err != nil {
			err := errors.Wrap(err, ErrJwt)
			span.RecordError(err)
			return err
		}
		hasAccess, err := a.evaluateBoolean(ctx, "data.opa.evaluation.can_create_employee",
			data.PolicyEvaluationInput{
				TokenUserId:         claims.UserId,
				PolicyCompileOutput: a.compileOutput.Load(),
			})
		if err != nil {
			span.RecordError(err)
			return err
		}
		if !hasAccess {
			err := ErrUnauthorized(claims.UserId, "delete",
				"employee", new(fmt.Sprint(empNo)))
			span.RecordError(err)
			return err
		}
	}
	if err := a.Logic.EmployeeDelete(ctx, empNo); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}
