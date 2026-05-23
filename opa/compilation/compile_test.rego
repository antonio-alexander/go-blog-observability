package opa.compilation_test

employees := {
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
}

roles := {
	"users_create": {
		"permissions": ["user_create"],
		"userIds": ["user_user_create"],
	},
	"users_read": {
		"permissions": ["user_read"],
		"userIds": ["user_user_read"],
	},
	"users_update": {
		"permissions": ["user_update"],
		"userIds": ["user_user_update"],
	},
	"users_delete": {
		"permissions": ["user_delete"],
		"userIds": ["user_user_delete"],
	},
	"employees_create": {
		"permissions": ["employee_create"],
		"userIds": ["user_employee_create"],
	},
	"employees_read": {
		"permissions": ["employee_read"],
		"userIds": ["user_employee_read"],
	},
	"employees_update": {
		"permissions": ["employee_update"],
		"userIds": ["user_employee_update"],
	},
	"employees_delete": {
		"permissions": ["employee_delete"],
		"userIds": ["user_employee_delete"],
	},
}

test_employee_read if {
	# Assert 'user_employee_read" is within employee read
	"user_employee_read" in data.opa.compilation.employees_read with input.roles as roles
}

test_employee_update if {
	# Assert 'user_employee_update" is within employee update
	"user_employee_update" in data.opa.compilation.employees_update with input.roles as roles
}

test_employee_delete if {
	# Assert 'user_employee_delete" is within employee delete
	"user_employee_delete" in data.opa.compilation.employees_delete with input.roles as roles
}

test_access if {
	# Assert that access has expected users populated
	access := data.opa.compilation.access with input.roles as roles
	"user_employee_create" in access.employees.create
	"user_employee_read" in access.employees.read
	"user_employee_update" in access.employees.update
	"user_employee_delete" in access.employees.delete
}

test_access_employees_create_only if {
	# Assert that if roles only exist for employees_create exist, the other entries are empty
	access := data.opa.compilation.access with input.roles as {"employees_create": {
		"permissions": ["employee_create"],
		"userIds": ["user_employee_create"],
	}}
	"user_employee_create" in access.employees.create
	count(access.employees.read) == 0
	count(access.employees.update) == 0
	count(access.employees.delete) == 0
}

test_access_employees_read_only if {
	# Assert that if roles only exist for employees_read exist, the other entries are empty
	access := data.opa.compilation.access with input.roles as {"employees_read": {
		"permissions": ["employee_read"],
		"userIds": ["user_employee_read"],
	}}
	count(access.employees.create) == 0
	"user_employee_read" in access.employees.read
	count(access.employees.update) == 0
	count(access.employees.delete) == 0
}

test_access_employees_update_only if {
	# Assert that if roles only exist for employees_update exist, the other entries are empty
	access := data.opa.compilation.access with input.roles as {"employees_update": {
		"permissions": ["employee_update"],
		"userIds": ["user_employee_update"],
	}}
	count(access.employees.create) == 0
	count(access.employees.read) == 0
	"user_employee_update" in access.employees.update
	count(access.employees.delete) == 0
}

test_access_employees_delete_only if {
	# Assert that if roles only exist for employees_delete exist, the other entries are empty
	access := data.opa.compilation.access with input.roles as {"employees_delete": {
		"permissions": ["employee_delete"],
		"userIds": ["user_employee_delete"],
	}}
	count(access.employees.create) == 0
	count(access.employees.read) == 0
	count(access.employees.update) == 0
	"user_employee_delete" in access.employees.delete
}

test_resources if {
	resources := data.opa.compilation.resources with input.roles as roles
		with input.employees as employees
	"user_employee_create" in resources.access.employees.create
	"user_employee_read" in resources.access.employees.read
	"user_employee_update" in resources.access.employees.update
	"user_employee_delete" in resources.access.employees.delete
	# some employee in employees
	# employee in resources.employees
}
