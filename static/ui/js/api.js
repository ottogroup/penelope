function bucketsProject(projectId, onDone, onFail) {
    $.ajax({
        url: "/api/buckets/" + projectId,
        type: "GET",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    }).fail(function (jqXHR, textStatus, errorThrown) {
        if (typeof onFail === "function") {
            onFail(jqXHR, textStatus, errorThrown)
        }
    });
}

function datasetsProject(projectId, onDone, onFail) {
    $.ajax({
        url: "/api/datasets/" + projectId,
        type: "GET",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    }).fail(function (jqXHR, textStatus, errorThrown) {
        if (typeof onFail === "function") {
            onFail(jqXHR, textStatus, errorThrown)
        }
    });
}

function createBackup(requestBackup, onDone, onFail) {
    $.ajax({
        url: "/api/backups",
        data: JSON.stringify(requestBackup),
        type: "POST",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    }).fail(function (jqXHR, textStatus, errorThrown) {
        if (typeof onFail === "function") {
            onFail(jqXHR, textStatus, errorThrown)
        }
    });
}

function calculateBackup(requestBackup, onDone, onFail) {
    $.ajax({
        url: "/api/backups/calculate",
        data: JSON.stringify(requestBackup),
        type: "POST",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    }).fail(function (jqXHR, textStatus, errorThrown) {
        if (typeof onFail === "function") {
            onFail(jqXHR, textStatus, errorThrown)
        }
    });
}

function getBackup(backupId, jobsPage, jobsPerPage, onDone, onFail) {
    if (!backupId) {
        return;
    }
    let url = "/api/backups/" + backupId;
    url += "?page=" + jobsPage.toString();
    if (jobsPerPage != null) {
        url += "&size=" + jobsPerPage.toString()
    }
    $.ajax({
        url: url,
        type: "GET",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    }).fail(function (jqXHR, textStatus, errorThrown) {
        if (typeof onFail === "function") {
            onFail(jqXHR, textStatus, errorThrown)
        }
    });
}

function resumeBackup(backupId, onDone, onFail) {
    updateBackup(backupId, "NotStarted", onDone, onFail);
}

function cancelBackup(backupId, onDone, onFail) {
    updateBackup(backupId, "Paused", onDone, onFail);
}

function deleteBackup(backupId, onDone, onFail) {
    updateBackup(backupId, "ToDelete", onDone, onFail);
}

function updateBackup(backupId, newStatus, onDone, onFail) {
    if (!backupId) {
        return;
    }
    var updateRequest = {};
    updateRequest.status = newStatus;
    updateRequest.backup_id = backupId
    $.ajax({
        url: "/api/backups/" + backupId,
        data: JSON.stringify(updateRequest),
        type: "PATCH",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    }).fail(function (jqXHR, textStatus, errorThrown) {
        if (typeof onFail === "function") {
            onFail(jqXHR, textStatus, errorThrown)
        }
    });
}

function configureBackup(backupId, configureRequest, onDone, onFail) {
    if (!backupId) {
        return;
    }
    $.ajax({
        url: "/api/backups/" + backupId,
        data: JSON.stringify(configureRequest),
        type: "PATCH",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    }).fail(function (jqXHR, textStatus, errorThrown) {
        if (typeof onFail === "function") {
            onFail(jqXHR, textStatus, errorThrown)
        }
    });
}

function listBackups(projectId, onDone, onFail) {
    var query = "";
    if (projectId) {
        query = "?project=" + projectId
    }

    $.ajax({
        url: "/api/backups" + query,
        type: "GET",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    }).fail(function (jqXHR, textStatus, errorThrown) {
        if (typeof onFail === "function") {
            onFail(jqXHR, textStatus, errorThrown)
        }
    });
}

function restoreBackup(backupId, jobID, onDone) {
    if (!backupId) {
        return;
    }
    var jobIDQuery = "";
    if (jobID != null) {
        jobIDQuery += "?jobIDForTimestamp=" + jobID;
    }
    $.ajax({
        url: "/api/restore/" + backupId + jobIDQuery,
        type: "GET",
        contentType: "application/json"
    }).done(function (data) {
        onDone(data);
    });
}
