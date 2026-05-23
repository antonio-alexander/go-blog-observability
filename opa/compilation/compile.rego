package opa.compilation

import rego.v1

# METADATA
# entrypoint: true
resources := {"access": access}

access := {"employees": {
	"create": employees_create,
	"read": employees_read,
	"update": employees_update,
	"delete": employees_delete,
}}

employees_create contains user_id if {
	some role in input.roles
	"employee_create" in role.permissions
	some user_id in role.userIds
}

employees_read contains user_id if {
	some role in input.roles
	"employee_read" in role.permissions
	some user_id in role.userIds
}

employees_update contains user_id if {
	some role in input.roles
	"employee_update" in role.permissions
	some user_id in role.userIds
}

employees_delete contains user_id if {
	some role in input.roles
	"employee_delete" in role.permissions
	some user_id in role.userIds
}
