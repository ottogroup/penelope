package model

// Role for user
type Role string

var (
    // None role for User doesn't grant any rights
    None Role = "none"
    // Viewer role for User grants view and listing for backups
    Viewer Role = "viewer"
    // Owner role for User grant all rights to the backups
    Owner Role = "owner"
)

func (r Role) String() string {
    return string(r)
}

// User object
type User struct {
    Email string
}

// Principal the principal of a request
type Principal struct {
    User         User
    RoleBindings []ProjectRoleBinding `yaml:"role_bindings"`
}


// ProjectRoleBinding defines User role bindings with the project
type ProjectRoleBinding struct {
    Role    Role
    Project string
}
