package data

type PolicyPermission string

const (
	PolicyPermissionEmployeeCreate PolicyPermission = "employee_create"
	PolicyPermissionEmployeeRead   PolicyPermission = "employee_read"
	PolicyPermissionEmployeeUpdate PolicyPermission = "employee_update"
	PolicyPermissionEmployeeDelete PolicyPermission = "employee_delete"
)

type PolicyRole struct {
	Name        string             `json:"name"`
	UserIds     []string           `json:"userIds"`
	Permissions []PolicyPermission `json:"permissions"`
}

type PolicyEmployee struct {
	Id     string `json:"id"`
	UserId string `json:"userId"`
}

type PolicyCompileInput struct {
	Roles map[string]PolicyRole `json:"roles"`
}

type PolicyCompileOutput struct {
	Access struct {
		Employees struct {
			Create []string `json:"create"`
			Read   []string `json:"read"`
			Update []string `json:"update"`
			Delete []string `json:"delete"`
		} `json:"employees"`
	} `json:"access"`
}

type PolicyEvaluationInput struct {
	TokenUserId string   `json:"tokenUserId"`
	EmployeeId  string   `json:"employeeId"`
	EmployeeIds []string `json:"employeeIds"`
	*PolicyCompileOutput
}

type PolicyEvaluateOutput struct {
	CanCreateEmployee bool `json:"can_create_employee"`
	CanReadEmployee   bool `json:"can_read_employee"`
	CanUpdateEmployee bool `json:"can_update_employee"`
	CanDeleteEmployee bool `json:"can_delete_employee"`
}
