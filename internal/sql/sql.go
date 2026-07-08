package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"

	"github.com/XSAM/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	_ "github.com/go-sql-driver/mysql" //import for driver support
)

const tableEmployees = "employees"

type mySql struct {
	logger.Logger
	tracer.Tracer
	config struct {
		Hostname       string        `json:"hostname"`
		Port           string        `json:"port"`
		Username       string        `json:"username"`
		Password       string        `json:"password"`
		Database       string        `json:"database"`
		ConnectTimeout time.Duration `json:"connect_timeout"`
		QueryTimeout   time.Duration `json:"query_timeout"`
		ParseTime      bool          `json:"parse_time"`
	}
	*sql.DB
	opened atomic.Bool
}

func New(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Sql
} {
	m := &mySql{}
	for _, parameter := range parameters {
		switch v := parameter.(type) {
		case logger.Logger:
			m.Logger = v
		case tracer.Tracer:
			m.Tracer = v
		}
	}
	return m
}

func (s *mySql) Configure(envs map[string]string) error {
	if databaseHost := envs["DATABASE_HOST"]; databaseHost != "" {
		s.config.Hostname = databaseHost
	}
	if databasePort := envs["DATABASE_PORT"]; databasePort != "" {
		s.config.Port = databasePort
	}
	if database := envs["DATABASE_NAME"]; database != "" {
		s.config.Database = database
	}
	if username := envs["DATABASE_USER"]; username != "" {
		s.config.Username = username
	}
	if password := envs["DATABASE_PASSWORD"]; password != "" {
		s.config.Password = password
	}
	if _, ok := envs["DATABASE_QUERY_TIMEOUT"]; ok {
		i, _ := strconv.ParseInt(envs["DATABASE_QUERY_TIMEOUT"], 10, 64)
		s.config.QueryTimeout = time.Duration(i) * time.Second
	}
	if _, ok := envs["DATABASE_PARSE_TIME"]; ok {
		s.config.ParseTime, _ = strconv.ParseBool(envs["DATABASE_PARSE_TIME"])
	}
	return nil
}

func (s *mySql) Open(ctx context.Context) error {
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=%t",
		s.config.Username, s.config.Password, s.config.Hostname,
		s.config.Port, s.config.Database, s.config.ParseTime)
	db, err := otelsql.Open("mysql", dataSourceName,
		otelsql.WithAttributes(semconv.DBSystemMySQL),
		// otelsql.WithMeterProvider(),
		// otelsql.WithTracerProvider(),
		otelsql.WithSpanOptions(otelsql.SpanOptions{Ping: false}),
	)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	s.DB = db
	s.opened.Store(true)
	return nil
}

func (s *mySql) Close(ctx context.Context) {
	if !s.opened.Load() {
		return
	}
	if err := s.DB.Close(); err != nil {
		s.Error(ctx, "unable to close sql", err)
	}
}

func (s *mySql) EmployeeCreate(ctx context.Context, employeePartial data.EmployeePartial) (*data.Employee, error) {
	var columns, values []string
	var args []any

	ctx, span := s.Start(ctx, "sql.employee_create")
	defer span.End()
	if employeePartial.BirthDate != nil {
		args = append(args, time.Unix(*employeePartial.BirthDate, 0))
		columns = append(columns, "birth_date")
		values = append(values, "?")
	}
	if employeePartial.FirstName != nil {
		args = append(args, employeePartial.FirstName)
		columns = append(columns, "first_name")
		values = append(values, "?")
	}
	if employeePartial.LastName != nil {
		args = append(args, employeePartial.LastName)
		columns = append(columns, "last_name")
		values = append(values, "?")
	}
	if employeePartial.Gender != nil {
		args = append(args, employeePartial.Gender)
		columns = append(columns, "gender")
		values = append(values, "?")
	}
	if employeePartial.HireDate != nil {
		args = append(args, time.Unix(*employeePartial.HireDate, 0))
		columns = append(columns, "hire_date")
		values = append(values, "?")
	}
	empNo, err := findEmpNo(ctx, s.DB)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	args = append(args, empNo)
	columns = append(columns, "emp_no")
	values = append(values, "?")
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableEmployees,
		strings.Join(columns, ","), strings.Join(values, ","))
	if _, err := s.ExecContext(ctx, query, args...); err != nil {
		span.RecordError(err)
		return nil, err
	}
	employee, err := s.EmployeeRead(ctx, empNo)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return employee, nil
}

func (s *mySql) EmployeeRead(ctx context.Context, empNo int64) (*data.Employee, error) {
	ctx, span := s.Start(ctx, "sql.employee_read")
	defer span.End()
	query := fmt.Sprintf(`SELECT emp_no, birth_date, first_name, last_name,
		gender, hire_date FROM %s WHERE emp_no = ?;`,
		tableEmployees)
	row := s.QueryRowContext(ctx, query, empNo)
	employee, err := employeeScan(row.Scan)
	if err != nil {
		switch {
		default:
			return nil, err
		case errors.Is(err, sql.ErrNoRows):
			err := ErrEmployeeNotFound(err, empNo)
			span.RecordError(err)
			return nil, err
		}
	}
	return employee, nil
}

func (s *mySql) EmployeesSearch(ctx context.Context, search data.EmployeeSearch) ([]*data.Employee, error) {
	var employees []*data.Employee

	ctx, span := s.Start(ctx, "sql.employee_search")
	defer span.End()
	criteria, args := employeeCriteria(search)
	query := fmt.Sprintf(`SELECT emp_no, birth_date, first_name, last_name,
		gender, hire_date FROM %s %s;`,
		tableEmployees, criteria)
	rows, err := s.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	for rows.Next() {
		employee, err := employeeScan(rows.Scan)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		employees = append(employees, employee)
	}
	if len(employees) <= 0 {
		span.RecordError(ErrEmployeeSearchNotFound)
		return nil, ErrEmployeeSearchNotFound
	}
	return employees, nil
}

func (s *mySql) EmployeeUpdate(ctx context.Context, empNo int64, employeePartial data.EmployeePartial) (*data.Employee, error) {
	var args []any
	var updates []string

	ctx, span := s.Start(ctx, "sql.employee_update")
	defer span.End()
	if employeePartial.BirthDate != nil {
		args = append(args, time.Unix(*employeePartial.BirthDate, 0))
		updates = append(updates, "birth_date = ?")
	}
	if employeePartial.FirstName != nil {
		args = append(args, employeePartial.FirstName)
		updates = append(updates, "first_name = ?")
	}
	if employeePartial.LastName != nil {
		args = append(args, employeePartial.LastName)
		updates = append(updates, "last_name = ?")
	}
	if employeePartial.Gender != nil {
		args = append(args, employeePartial.Gender)
		updates = append(updates, "gender =  ?")
	}
	if employeePartial.HireDate != nil {
		args = append(args, time.Unix(*employeePartial.HireDate, 0))
		updates = append(updates, "hire_date = ?")
	}
	query := fmt.Sprintf("UPDATE %s SET %s WHERE emp_no = ?", tableEmployees,
		strings.Join(updates, ","))
	args = append(args, empNo)
	if _, err := s.ExecContext(ctx, query, args...); err != nil {
		span.RecordError(err)
		return nil, err
	}
	employee, err := s.EmployeeRead(ctx, empNo)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return employee, nil
}

func (s *mySql) EmployeeDelete(ctx context.Context, empNo int64) error {
	ctx, span := s.Start(ctx, "sql.employee_delete")
	defer span.End()
	query := fmt.Sprintf(`DELETE FROM %s WHERE emp_no = ?;`,
		tableEmployees)
	result, err := s.ExecContext(ctx, query, empNo)
	if err != nil {
		span.RecordError(err)
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if n == 0 {
		err := ErrEmployeeNotFound(nil, empNo)
		span.RecordError(err)
		return err
	}
	return nil
}

func (s *mySql) Sleep(ctx context.Context, sleep data.Sleep) (*data.Sleep, error) {
	ctx, span := s.Start(ctx, "sql.sleep")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()
	query := "DO SLEEP(?);"
	if _, err := s.ExecContext(ctx, query, sleep.Duration); err != nil {
		span.RecordError(err)
		return nil, err
	}
	return &sleep, nil
}
