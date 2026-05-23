package opa.evaluation

user_id_from_employee_id(employee_id) := user_id if {
	employee := input.employees[employee_id]
	employee != null
	user_id = employee.userId
}

# METADATA
# entrypoint: true
default can_create_employee := false

can_create_employee if {
	input.employeeUserId != ""
	input.tokenUserId == input.employeeUserId
}

can_create_employee if {
	input.tokenUserId != ""
	input.tokenUserId in input.access.employees.create
}

can_create_employee if {
	input.employeeUserId == ""
	input.employeeId != ""
	employee_user_id := user_id_from_employee_id(input.employeeId)
	input.tokenUserId == employee_user_id
	employee_user_id == input.employeeUserId
}

# METADATA
# entrypoint: true
default can_read_employee := false

can_read_employee if {
	input.tokenUserId != ""
	input.employeeId != ""
	employee_user_id := user_id_from_employee_id(input.employeeId)
	input.tokenUserId == employee_user_id
}

can_read_employee if {
	input.tokenUserId != ""
	input.tokenUserId in input.access.employees.read
}

can_read_employee if {
	can_create_employee
}

can_read_employee if {
	can_update_employee
}

can_read_employee if {
	can_delete_employee
}

# METADATA
# entrypoint: true
default can_update_employee := false

can_update_employee if {
	input.tokenUserId != ""
	input.employeeId != ""
	employee_user_id := user_id_from_employee_id(input.employeeId)
	input.tokenUserId == employee_user_id
}

can_update_employee if {
	input.tokenUserId != ""
	input.tokenUserId in input.access.employees.update
}

can_update_employee if {
	can_create_employee
}

# METADATA
# entrypoint: true
default can_delete_employee := false

can_delete_employee if {
	input.tokenUserId != ""
	input.employeeId != ""
	employee_user_id := user_id_from_employee_id(input.employeeId)
	input.tokenUserId == employee_user_id
}

can_delete_employee if {
	input.tokenUserId != ""
	input.tokenUserId in input.access.employees.delete
}

can_delete_employee if {
	can_create_employee
}

user_employees[employee.id] := employee if {
	can_read_employee
	some employee in input.employees
}

user_employees[employee.id] := employee if {
	some employee in input.employees

	# this could be fixed by making can_read_employee a function
	# but that would harm readability and increase complexity
	# significantly for not much gain; issues with caching
	# rule outputs don't matter here
	# regal ignore:with-outside-test-context
	can_read_employee with input.employeeId as employee.id
}

user_employees_filter[employee.id] := employee if {
	can_read_employee
	some employee in input.employees
	employee.id in input.employeeIds
}

user_employees_filter[employee.id] := employee if {
	some employee in input.employees
	employee.id in input.employeeIds

	# this could be fixed by making can_read_employee a function
	# but that would harm readability and increase complexity
	# significantly for not much gain; issues with caching
	# rule outputs don't matter here
	# regal ignore:with-outside-test-context
	can_read_employee with input.employeeId as employee.id
}
