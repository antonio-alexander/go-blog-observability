package data

const (
	RouteCache           string = "/cache"
	RouteSleep           string = "/sleep"
	RoutePanic           string = "/panic"
	RouteAuthorize       string = "/authorize"
	RouteEmployees       string = "/employees"
	RouteEmployeesSearch string = RouteEmployees + "/search"
	RouteEmployeesEmpNo  string = RouteEmployees + "/{" + PathEmpNo + "}"
	RouteEmployeesEmpNof string = RouteEmployees + "/%d"
)

const (
	PathEmpNo       string = "EmpNo"
	ParameterEmpNos string = "emp_nos"
)

type Request struct {
	EmployeePartial EmployeePartial `json:"employee_partial"`
	UserId          string          `json:"user_id"`
}

type Response struct {
	Employee      *Employee   `json:"employee,omitempty"`
	Employees     []*Employee `json:"employees,omitempty"`
	Authorization *string     `json:"authorization,omitempty"`
}
