package auth

import (
    "github.com/ottogroup/penelope/pkg/http/auth/model"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "reflect"
)

// CheckRequestIsAllowed check if user is allowed to perform request
func CheckRequestIsAllowed(principal *model.Principal, requestType requestobjects.RequestType, project string) bool {
    if principal == nil || reflect.ValueOf(principal).IsNil() || principal.User.Email == "" {
        return false
    }

    isAllowed := false
    rbacRole := userRoleInProject(principal, project)
    switch requestType {
    case requestobjects.Updating:
        isAllowed = matchRole(rbacRole, model.Owner)
    case requestobjects.Creating:
        isAllowed = matchRole(rbacRole, model.Owner)
    case requestobjects.Getting, requestobjects.Listing, requestobjects.Restoring, requestobjects.Calculating,
        requestobjects.DatasetListing, requestobjects.BucketListing:
        isAllowed = matchRole(rbacRole, model.Owner, model.Viewer)
    }

    return isAllowed
}

func userRoleInProject(principal *model.Principal, project string) model.Role {
    for _, projectRole := range principal.RoleBindings {
        if projectRole.Project == project {
            return projectRole.Role
        }
    }
    return model.None
}

func matchRole(userRole model.Role, allowedRoles ...model.Role) bool {
    for _, role := range allowedRoles {
        if userRole == role {
            return true
        }
    }
    return false
}
