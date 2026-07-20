package authz

import "github.com/QuantumNous/new-api/common"

// resolveSubjectRoles returns the role keys assigned to a subject. The mapping
// is derived from the caller's system role.
var resolveSubjectRoles = func(userID int, systemRole int) []string {
	switch {
	case systemRole >= common.RoleRootUser:
		return []string{BuiltInRoleRoot}
	case systemRole >= common.RoleAdminUser:
		return []string{BuiltInRoleAdmin}
	case systemRole >= common.RoleAuthorizedAdmin:
		return []string{BuiltInRoleAuthorizedAdmin}
	default:
		return nil
	}
}

func managedRoleKeyForSystemRole(systemRole int) (string, bool) {
	switch {
	case systemRole == common.RoleAuthorizedAdmin:
		return BuiltInRoleAuthorizedAdmin, true
	case systemRole == common.RoleAdminUser:
		return BuiltInRoleAdmin, true
	default:
		return "", false
	}
}
