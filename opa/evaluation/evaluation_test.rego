package opa.evaluation_test

compiled_policy_data := {
	"employees": {
		"employee_create": {
			"id": "employee_create",
			"userId": "user_employee_create",
		},
		"employee_read": {
			"id": "employee_read",
			"userId": "user_employee_read",
		},
		"employee_update": {
			"id": "employee_update",
			"userId": "user_employee_update",
		},
		"employee_delete": {
			"id": "employee_delete",
			"userId": "user_employee_delete",
		},
		"employee_nobody": {
			"id": "employee_nobody",
			"userId": "user_employee_nobody",
		},
	},
	"access": {"employees": {
		"create": ["user_employee_create"],
		"read": ["user_employee_read"],
		"update": ["user_employee_update"],
		"delete": ["user_employee_delete"],
	}},
}

test_can_create_employees_grant if {
	data.opa.evaluation.can_create_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_create"
	data.opa.evaluation.can_create_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_nobody"
		with input.employeeUserId as "user_employee_nobody"
}

test_can_create_employee_deny if {
	not data.opa.evaluation.can_create_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_read"
		with input.employeeUserId as "user_employee_read"
	not data.opa.evaluation.can_create_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_update"
		with input.employeeUserId as "user_employee_update"
	not data.opa.evaluation.can_create_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_delete"
		with input.employeeUserId as "user_employee_delete"
	not data.opa.evaluation.can_create_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_nobody"
		with input.employeeUserId as "user_employee_nobody_"
	not data.opa.evaluation.can_create_employee with input.access as compiled_policy_data.access
}

test_can_read_employee_grant if {
	data.opa.evaluation.can_read_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_read"
	data.opa.evaluation.can_read_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_create"
	data.opa.evaluation.can_read_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_update"
	data.opa.evaluation.can_read_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_delete"
	data.opa.evaluation.can_read_employee with input.access as compiled_policy_data.access
		with input.employees as compiled_policy_data.employees
		with input.tokenUserId as "user_employee_nobody"
		with input.employeeId as "employee_nobody"
}

test_can_read_employee_deny if {
	not data.opa.evaluation.can_read_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_nobody"
	not data.opa.evaluation.can_read_employee with input.access as compiled_policy_data.access
		with input.employees as compiled_policy_data.employees
		with input.tokenUserId as "user_employee_nobody"
		with input.employeeId as "employee_create"
	not data.opa.evaluation.can_read_employee with input.access as compiled_policy_data.access
}

test_can_update_employee_grant if {
	data.opa.evaluation.can_update_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_update"
	data.opa.evaluation.can_update_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_create"
	data.opa.evaluation.can_update_employee with input.access as compiled_policy_data.access
		with input.employees as compiled_policy_data.employees
		with input.tokenUserId as "user_employee_nobody"
		with input.employeeId as "employee_nobody"
}

test_can_update_employee_deny if {
	not data.opa.evaluation.can_update_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_user_read"
	not data.opa.evaluation.can_update_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_user_delete"
	not data.opa.evaluation.can_update_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_user_nobody"
	not data.opa.evaluation.can_update_employee with input.access as compiled_policy_data.access
		with input.employees as compiled_policy_data.employees
		with input.tokenUserId as "user_employee_nobody"
		with input.employeeId as "employee_create"
	not data.opa.evaluation.can_update_employee with input.access as compiled_policy_data.access
}

test_can_delete_employee_grant if {
	data.opa.evaluation.can_delete_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_delete"
	data.opa.evaluation.can_delete_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_create"
	data.opa.evaluation.can_delete_employee with input.access as compiled_policy_data.access
		with input.employees as compiled_policy_data.employees
		with input.tokenUserId as "user_employee_nobody"
		with input.employeeId as "employee_nobody"
}

test_can_delete_employee_deny if {
	not data.opa.evaluation.can_delete_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_read"
	not data.opa.evaluation.can_delete_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_update"
	not data.opa.evaluation.can_delete_employee with input.access as compiled_policy_data.access
		with input.tokenUserId as "user_employee_nobody"
	not data.opa.evaluation.can_delete_employee with input.access as compiled_policy_data.access
}

test_user_employees_explicit if {
	employees := data.opa.evaluation.user_employees with input.tokenUserId as "user_employee_read"
		with input.access as compiled_policy_data.access
		with input.employees as compiled_policy_data.employees
	count(employees) == count(compiled_policy_data.employees)
	some employee in employees
	employee in compiled_policy_data.employees
}

test_user_employees_implicit if {
	employees := data.opa.evaluation.user_employees with input.tokenUserId as "user_employee_nobody"
		with input.access as compiled_policy_data.access
		with input.employees as compiled_policy_data.employees
	count(employees) == 1
	compiled_policy_data.employees.employee_nobody in employees
}

test_user_employees_filter if {
	employee_ids := ["employee_nobody"]
	employees := data.opa.evaluation.user_employees_filter with input.tokenUserId as "user_employee_read"
		with input.employeeIds as employee_ids
		with input.access as compiled_policy_data.access
		with input.employees as compiled_policy_data.employees
	count(employee_ids) == count(employees)
	some employee in employees
	employee.id in employee_ids
	employee in compiled_policy_data.employees
}
