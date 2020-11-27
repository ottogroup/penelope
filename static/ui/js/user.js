var User = (function () {
    function User(me) {
        this.me = me;
    }

    User.prototype.projects = function () {
        var projects = [];
        if (!this.me || !this.me.RoleBindings) {
            return projects;
        }
        for (i = 0; i < this.me.RoleBindings.length; i++) {
            projects.push(this.me.RoleBindings[i].Project);
        }
        return projects.sort();
    };
    User.prototype.projectsWithRoles = function (roles) {
        var projects = [];
        if (!this.me || !this.me.RoleBindings || !roles.length || roles.length < 1) {
            return projects;
        }
        for (i = 0; i < this.me.RoleBindings.length; i++) {
            let project = this.me.RoleBindings[i].Project;
            let projectRole = this.me.RoleBindings[i].Role;
            for (j = 0; j < roles.length; j++) {
                if (0 == projectRole.localeCompare(roles[j])) {
                    projects.push(project);
                    break;
                }
            }
        }
        return projects.sort();
    };
    return User;
})();
