function initBootstrap() {
    // return $.fn.button to previously assigned value
    // give $().bootstrapBtn the Bootstrap functionality
    $.fn.bootstrapBtn = $.fn.button.noConflict();
}

var App = (function () {
    const rowsPerPage = [50, 100, 200, 500, 1000];

    function requestBackupFromForm() {
        let requestBackup = {};
        requestBackup.type = $("#create-backup-form-type").val();
        requestBackup.strategy = $("#create-backup-form-strategy").val();
        requestBackup.project = $("#create-backup-form-project").val();
        requestBackup.target = {
            region: $("#create-backup-form-storage-region").val(),
            storage_class: $("#create-backup-form-storage-class").val()
        };
        let archive_ttm = parseInt($("#update-backup-form-archive-ttm").val());
        if (archive_ttm >= 0) {
            requestBackup.target.archive_ttm = archive_ttm
        }

        requestBackup.snapshot_options = {};
        if (requestBackup.strategy === "Snapshot" || requestBackup.strategy === "Oneshot") {
            let snapshot_ttl = parseInt($("#create-backup-form-snapshot-ttl").val());
            if (snapshot_ttl > 0) {
                requestBackup.snapshot_options.lifetime_in_days = snapshot_ttl;
            }
            let snapshot_schedule = parseInt($("#create-backup-form-snapshot-schedule").val());
            if (requestBackup.strategy === "Oneshot") {
                requestBackup.strategy = "Snapshot"
                requestBackup.snapshot_options.frequency_in_hours = 0;
            } else if (snapshot_schedule > 0) {
                requestBackup.snapshot_options.frequency_in_hours = snapshot_schedule;
            }
        }

        requestBackup.mirror_options = {};
        if (requestBackup.strategy === "Mirror") {
            let mirror_ttl = parseInt($("#create-backup-form-mirror-ttl").val());
            if (mirror_ttl > 0) {
                requestBackup.mirror_options.lifetime_in_days = mirror_ttl;
            }

        }

        requestBackup.bigquery_options = {dataset: $("#create-backup-form-bigquery-dataset").val()};
        let tables = $("#create-backup-form-bigquery-table").val();
        if (0 < tables.length) {
            requestBackup.bigquery_options.table = tables.split(",")
        }
        let excluded_tables = $("#create-backup-form-bigquery-excluded_tables").val();
        if (0 < excluded_tables.length) {
            requestBackup.bigquery_options.excluded_tables = excluded_tables.split(",")
        }

        requestBackup.gcs_options = {bucket: $("#create-backup-form-storage-bucket").val()};
        let storageInclude = $("#create-backup-form-storage-include").val();
        if (0 < storageInclude.length) {
            requestBackup.gcs_options.include_prefixes = storageInclude.split(",");
        }
        let storageExclude = $("#create-backup-form-storage-exclude").val();
        if (0 < storageExclude.length) {
            requestBackup.gcs_options.exclude_prefixes = storageExclude.split(",");
        }
        return requestBackup;
    }

    function requestUpdateBackupFromForm() {
        let backupId = $("#update-backup-form-backup-id").val();
        let requestBackup = {backup_id: backupId};
        requestBackup.table = "";
        let tables = $("#update-backup-form-bigquery-table").val();
        if (0 < tables.length) {
            requestBackup.table = tables.split(",")
        }

        requestBackup.excluded_tables = "";
        let excluded_tables = $("#update-backup-form-bigquery-excluded_tables").val();
        if (0 < excluded_tables.length) {
            requestBackup.excluded_tables = excluded_tables.split(",")
        }
        requestBackup.include_path = "";
        requestBackup.exclude_path = "";
        let storageInclude = $("#update-backup-form-storage-include").val();
        if (0 < storageInclude.length) {
            requestBackup.include_path = storageInclude.split(",");
        }
        let mirror_ttl = parseInt($("#update-backup-form-mirror-ttl").val());
        if (mirror_ttl >= 0) {
            requestBackup.mirror_ttl = mirror_ttl;
        }
        let snapshot_ttl = parseInt($("#update-backup-form-snapshot-ttl").val());
        if (snapshot_ttl >= 0) {
            requestBackup.snapshot_ttl = snapshot_ttl;
        }
        let archive_ttm = parseInt($("#update-backup-form-archive-ttm").val());
        if (archive_ttm >= 0) {
            requestBackup.archive_ttm = archive_ttm;
        }

        let storageExclude = $("#update-backup-form-storage-exclude").val();
        if (0 < storageExclude.length) {
            requestBackup.exclude_path = storageExclude.split(",");
        }
        return requestBackup;
    }

    function App() {
        var that = this;

        this.infoSnackBar = mdc.snackbar.MDCSnackbar.attachTo(document.querySelector('.mdc-snackbar.info'));
        this.infoSnackBar.timeoutMs = 10000;

        function calculateSuccess(data) {
            closeInfo();
            let dataSize = 0;
            let chartData = [];
            let chartLabels = [];
            data.costs.forEach(function (item) {
                chartData.push({x: item.period, y: item.cost.toFixed(4)});
                chartLabels.push(item.period);
                dataSize = (item.size_in_bytes / Math.pow(1024, 3)).toFixed(2);
            });
            let ctxBox = document.getElementById('calculateChart');
            ctxBox.style.width = '360px';
            ctxBox.style.height = '180px';
            ctxBox.innerHTML = '';
            let canvas = document.createElement('canvas');
            ctxBox.appendChild(canvas);
            var myChart = new Chart(canvas, {
                type: 'line',
                layout: "fitColumns",
                data: {
                    datasets: [{
                        label: '€ at month for ' + dataSize + " GB",
                        data: chartData,
                        borderWidth: 1
                    }],
                    labels: chartLabels,
                },
                options: {
                    scales: {
                        xAxes: [{
                            ticks: {
                                beginAtZero: true
                            }
                        }],
                        yAxes: [{
                            ticks: {
                                beginAtZero: true,
                                callback: function (value, index, values) {
                                    return '€ ' + value.toFixed(2);
                                }
                            }
                        }]
                    }
                }
            });
        }

        function calculateFailed(jqXHR, textStatus, errorThrown) {
            openInfo("Backup: calculate " + errorThrown + ": " + jqXHR.responseText);
            let ctxBox = document.getElementById('calculateChart');
            ctxBox.style = '';
            ctxBox.innerHTML = '';
        }

        var openInfo = function (msg) {
            that.infoSnackBar.labelText = msg;
            that.infoSnackBar.open();
        };
        var closeInfo = function () {
            that.infoSnackBar.close();
            that.infoSnackBar.labelText = "";
        };
        // backup list
        const backupDialog = mdc.dialog.MDCDialog.attachTo(document.querySelector('.mdc-dialog.backup'));
        document.querySelector(".mdc-button.close_backup_jobs").addEventListener("click", function (event) {
            event.stopPropagation();
            backupDialog.close();
        }, {passive: true});
        backupDialog.listen('MDCDialog:closed', () => {
            let backupContent = document.getElementById("backup-jobs-content");
            backupContent.innerHtml = "";
        });

        // update
        function prepareUpdateBackupForm(backupId) {
            openInfo("Backup: Getting details for " + backupId);
            getBackup(backupId, 0, 0, function (backup) {
                // reset form
                document.querySelectorAll('.mdc-dialog.update_backup input').forEach(function (element) {
                    element.value = ""
                });
                // show form
                document.getElementById('update-backup-form-backup-id').value = backup.id;
                if (backup.type === "BigQuery") {
                    document.querySelectorAll('.bq').forEach(function (element) {
                        element.classList.remove('hidden')
                    });
                    document.querySelectorAll('.gcs').forEach(function (element) {
                        element.classList.add('hidden')
                    });
                    if (backup.bigquery_options && backup.bigquery_options.table && 0 < backup.bigquery_options.table.length) {
                        document.getElementById('update-backup-form-bigquery-table').value = backup.bigquery_options.table.join(',')
                    }
                    if (backup.bigquery_options && backup.bigquery_options.excluded_tables && 0 < backup.bigquery_options.excluded_tables.length) {
                        document.getElementById('update-backup-form-bigquery-excluded_tables').value = backup.bigquery_options.excluded_tables.join(',')
                    }
                }
                if (backup.type === "CloudStorage") {
                    document.querySelectorAll('.bq').forEach(function (element) {
                        element.classList.add('hidden')
                    });
                    document.querySelectorAll('.gcs').forEach(function (element) {
                        element.classList.remove('hidden')
                    });
                    if (backup.gcs_options) {
                        if (backup.gcs_options && backup.gcs_options.include_prefixes) {
                            document.getElementById('update-backup-form-storage-include').value = backup.gcs_options.include_prefixes
                        }
                        if (backup.gcs_options && backup.gcs_options.exclude_prefixes) {
                            document.getElementById('update-backup-form-storage-exclude').value = backup.gcs_options.exclude_prefixes
                        }
                    }
                }
                if (backup.strategy === "Mirror") {
                    document.querySelectorAll('.update-mirror').forEach(function (element) {
                        element.classList.remove('hidden')
                    });
                    if (backup.type === "BigQuery" && backup.mirror_options.lifetime_in_days) {
                        document.getElementById('update-backup-form-mirror-ttl').value = backup.mirror_options.lifetime_in_days
                    }
                    if (backup.type === "CloudStorage" && backup.mirror_options.lifetime_in_days) {
                        document.getElementById('update-backup-form-mirror-ttl').value = backup.mirror_options.lifetime_in_days
                    }
                }
                if (backup.strategy === "Snapshot" && backup.type === "CloudStorage") {
                    document.querySelectorAll('.update-snapshot').forEach(function (element) {
                        element.classList.remove('hidden')
                    });
                    if (backup.snapshot_options.lifetime_in_days) {
                        document.getElementById('update-backup-form-snapshot-ttl').value = backup.mirror_options.lifetime_in_days
                    }
                }
                if (backup.target && backup.target.archive_ttm >= 0) {
                    document.getElementById('update-backup-form-archive-ttm').value = backup.target.archive_ttm
                }
                // open dialog
                configureBackupDialog.open();
                configureBackupDialog.layout();
            })
        }

        function showBackup(backupId, jobsPage, jobsPerPage, quiet) {
            if (!quiet) {
                openInfo("Backup: Getting details for " + backupId);
            }
            if (!jobsPerPage || jobsPerPage < rowsPerPage[0]) {
                jobsPerPage = rowsPerPage[0];
            }
            getBackup(backupId, jobsPage, jobsPerPage, function (backup) {
                document.querySelector("#backup-content span.created").innerHTML = simpleDataTime(backup.created);
                if (backup.updated) {
                    document.querySelector("#backup-content span.updated").innerHTML = simpleDataTime(backup.updated);
                } else {
                    document.querySelector("#backup-content span.updated").innerHTML = "-";
                }
                let p_extra = document.querySelector("#backup-content p.snapshot");
                p_extra.innerHTML = "";
                let storageInfo = "Storage region: " + backup.target.region + "<br>";
                storageInfo += "Storage class: " + backup.target.storage_class + "<br>";
                let snapshotInfo = "";
                if (backup.strategy === "Snapshot" && backup.snapshot_options) {
                    if (backup.snapshot_options.frequency_in_hours) {
                        snapshotInfo += "Frequency in hours: " + backup.snapshot_options.frequency_in_hours + "<br>";
                    }
                    if (backup.snapshot_options.lifetime_in_days) {
                        snapshotInfo += "lifetime in days: " + backup.snapshot_options.lifetime_in_days;
                    }
                }
                let gcsInfo = "";
                if (backup.type === "CloudStorage" && backup.gcs_options) {
                    if (backup.gcs_options.include_prefixes && 0 < backup.gcs_options.include_prefixes.length) {
                        gcsInfo += "Include prefixes: " + backup.gcs_options.include_prefixes.join(", ") + "<br>";
                    }
                    if (backup.gcs_options.exclude_prefixes && 0 < backup.gcs_options.exclude_prefixes.length) {
                        gcsInfo += "Exclude prefixes: " + backup.gcs_options.exclude_prefixes.join(", ") + "<br>";
                    }
                }
                p_extra.innerHTML = storageInfo + snapshotInfo + gcsInfo;
                let bj_content = document.getElementById('backup-jobs-content');
                bj_content.innerHTML = "";
                let bj_table = document.createElement("table");
                bj_table.setAttribute("class", "mdl-data-table mdl-js-data-table mdl-shadow--2dp");
                let bj_thead = document.createElement("thead");
                bj_thead.innerHTML = '<tr>\n' +
                    '  <th class="mdl-data-table__cell--non-numeric">Status</th>\n' +
                    '  <th class="mdl-data-table__cell--non-numeric">Source</th>\n' +
                    '  <th class="mdl-data-table__cell--non-numeric">Updated</th>\n' +
                    '  <th class="mdl-data-table__cell--non-numeric">Foreign Job ID</th>\n' +
                    '  <th class="mdl-data-table__cell--non-numeric">Action</th>\n' +
                    '</tr>';
                bj_table.appendChild(bj_thead);
                if (backup.jobs) {
                    let bj_tbody = document.createElement("tbody");

                    for (i = 0; i < backup.jobs.length; i++) {
                        let bj_tr = document.createElement('tr');
                        let foreign_job_id = "";
                        if (backup.jobs[i].foreign_job_id) {
                            foreign_job_id = backup.jobs[i].foreign_job_id;
                        }
                        // TODO: handle other job statuses
                        let status = "";
                        if (backup.jobs[i].status === "Scheduled") {
                            status = "<i class=\"material-icons mdc-top-app-bar__navigation-icon\">schedule</i>"
                        } else if (backup.jobs[i].status === "FinishedOk") {
                            status = "<i class=\"material-icons mdc-top-app-bar__navigation-icon\">done</i>";
                        } else {
                            status = "<i class=\"material-icons mdc-top-app-bar__navigation-icon\">highlight_off</i>";
                        }
                        let bj_tr_innerHTML = '<td>' + status + '</td>\n' +
                            '<td class="mdl-data-table__cell--non-numeric">' + backup.jobs[i].source + '</td>\n' +
                            '<td class="mdl-data-table__cell--non-numeric">' + simpleDataTime(backup.jobs[i].updated) + '</td>\n' +
                            '<td class="mdl-data-table__cell--non-numeric">' + foreign_job_id + '</td>\n' +
                            '<td class="mdl-data-table__cell--non-numeric">';
                        if (backup.type === "BigQuery") {
                            bj_tr_innerHTML += '<a href="#" onclick="App.handleRestoreBackup(this,\'' + backup.id + '\',\'' + backup.jobs[i].id + '\')">show import bq</a>';
                        }
                        bj_tr_innerHTML += '</td>\n';
                        bj_tr.innerHTML = bj_tr_innerHTML;
                        bj_tbody.appendChild(bj_tr);
                    }
                    bj_table.appendChild(bj_tbody);
                }
                bj_content.appendChild(bj_table);

                // handle jobs pagination
                let pagination = document.createElement("div");
                let selectRowsPerPage = document.createElement("select");
                selectRowsPerPage.id = "backup-jobs-per-page";
                selectRowsPerPage.addEventListener("change", (ev) => {
                    // show the first page of jobs after changing how many items to fetch
                    showBackup(backupId, 0, parseInt(ev.srcElement.value), true);
                });
                for (i = 0; i < rowsPerPage.length; i++) {
                    let option = document.createElement("option");
                    option.value = rowsPerPage[i];
                    option.innerText = rowsPerPage[i].toString();
                    if (rowsPerPage[i] === jobsPerPage) {
                        option.selected = "selected";
                    }
                    selectRowsPerPage.appendChild(option)
                }
                pagination.appendChild(selectRowsPerPage);
                if (0 < backup.jobs_total && jobsPerPage < backup.jobs_total) {
                    if (jobsPage >= 1) {
                        let prev_page = __createPaginationButton(backupId, jobsPage - 1, jobsPerPage, "keyboard_arrow_left");
                        pagination.appendChild(prev_page);
                    } else {
                        let prev_page = __createPaginationButton(backupId, 0, 0, "keyboard_arrow_left", true);
                        pagination.appendChild(prev_page);
                    }
                    let curr_page = document.createElement("input");
                    curr_page.type = "hidden";
                    curr_page.value = jobsPage + 1;
                    pagination.appendChild(curr_page);

                    let curr_page_info = document.createElement("em");
                    curr_page_info.innerText = (jobsPage + 1).toString() + " of " + Math.ceil(backup.jobs_total / jobsPerPage).toString();
                    pagination.appendChild(curr_page_info);

                    if (jobsPage < (backup.jobs_total / jobsPerPage) - 1) {
                        let next_page = __createPaginationButton(backupId, jobsPage + 1, jobsPerPage, "keyboard_arrow_right");
                        pagination.appendChild(next_page);
                    } else {
                        let next_page = __createPaginationButton(backupId, jobsPage + 1, jobsPerPage, "keyboard_arrow_right", true);
                        pagination.appendChild(next_page);
                    }
                    let placeholder_for_pagination = document.querySelector("#backup-jobs-pagination");
                    placeholder_for_pagination.innerHTML = "";
                    placeholder_for_pagination.appendChild(pagination);
                }

                backupDialog.open();
            });
        }

        function __createPaginationButton(backupId, page, rowsPerPage, icon, disabled = false) {
            let button = document.createElement("button");
            if (disabled) {
                button.disabled = true;
            }
            button.classList.add("mdc-button");
            button.innerHTML = "<i class=\"material-icons\">" + icon + "</i>";
            button.addEventListener("click", (ev) => {
                showBackup(backupId, page, rowsPerPage, true);
            });
            return button
        }

        var table = new Tabulator("#backup-list", {
                layout: "fitData",
                columns: [
                    {
                        title: "",
                        field: "id",
                        align: "center",
                        headerSort: false,
                        width: 64,
                        formatter: function (cell, formatterParams, onRendered) {
                            let data = cell.getData();
                            let button = document.createElement("button");
                            button.setAttribute("class", "mdc-button");
                            button.setAttribute("id", data.id);
                            let htmlIcon = document.createElement("i");
                            htmlIcon.setAttribute("class", "material-icons");
                            if (cell.getRow().isSelected()) {
                                htmlIcon.innerHTML = "check_box";
                            } else {
                                htmlIcon.innerHTML = "crop_square";
                            }
                            button.appendChild(htmlIcon);
                            return button.outerHTML;
                        },
                        cellClick: function (e, cell) {
                            //e - the click event object
                            //cell - cell component
                            if (cell.getValue() === "") {
                                return;
                            }
                            let row = cell.getRow();
                            row.toggleSelect();
                        },
                    },
                    {
                        title: "",
                        field: "id",
                        align: "center",
                        headerSort: false,
                        width: 64,
                        formatter: function (cell, formatterParams, onRendered) {
                            let data = cell.getData();
                            let button = document.createElement("button");
                            button.setAttribute("class", "mdc-button update_backup");
                            button.setAttribute("id", data.id);
                            let htmlIcon = document.createElement("i");
                            htmlIcon.setAttribute("class", "material-icons mdc-button__icon");
                            htmlIcon.innerHTML = "build";
                            button.appendChild(htmlIcon);
                            return button.outerHTML;
                        },
                        cellClick: function (e, cell) {
                            //e - the click event object
                            //cell - cell component
                            let backupId = cell.getValue();
                            prepareUpdateBackupForm(backupId);
                        },
                    },
                    {
                        title: "",
                        field: "id",
                        align: "left",
                        headerSort: false,
                        width: 64,
                        formatter: function (cell, formatterParams, onRendered) {
                            let data = cell.getData();
                            let button = document.createElement("button");
                            button.setAttribute("class", "mdc-button");
                            button.setAttribute("id", data.id);
                            let htmlIcon = document.createElement("i");
                            htmlIcon.setAttribute("class", "material-icons mdc-button__icon");
                            htmlIcon.innerHTML = "list";
                            button.appendChild(htmlIcon);
                            return button.outerHTML;
                        },
                        cellClick: function (e, cell) {
                            //e - the click event object
                            //cell - cell component
                            let backupId = cell.getValue();
                            showBackup(backupId, 0, 50, false);
                        },
                    },
                    {
                        title: "type",
                        field: "type",
                        sorter: "string",
                        align: "left"
                    },
                    {
                        title: "project",
                        field: "project",
                        sorter: "string",
                        align: "left"
                    },
                    {title: "strategy", field: "strategy", sorter: "string", align: "left"},
                    {title: "status", field: "status", sorter: "string", align: "left"},
                    {
                        title: "sink bucket", field: "sink_url", align: "left", formatter: "link",
                        headerSort: false,
                        formatterParams: {
                            labelField: "sink",
                            target: "_blank",
                            url: function (rowComponent) {
                                data = rowComponent.getRow().getData();
                                return sinkBucketUrl(data.sink_project, data.sink);
                            }
                        }
                    },
                    {
                        title: "source",
                        sorter: "string",
                        field: "id",
                        headerSort: false,
                        align: "left",
                        formatter: function (cell, formatterParams, onRendered) {
                            let data = cell.getData();
                            let innerHtml = '';
                            if (data.type === "BigQuery") {
                                innerHtml += 'Dataset: <a href="HHH" target="_blank">'.replace('HHH', datasetUrl(data.project, data.bigquery_options.dataset)) + data.bigquery_options.dataset + '</a>';
                                innerHtml += '<ul>';
                                if (data.bigquery_options && data.bigquery_options.table && 0 < data.bigquery_options.table.length) {
                                    data.bigquery_options.table.forEach(function (table) {
                                        innerHtml += '<li>table: <a href="HHH" target="_blank">'.replace('HHH', tableUrl(data.project, data.bigquery_options.dataset, table)) + table + '</a></li>';
                                    });
                                }
                                innerHtml += '</ul>';
                            }
                            if (data.type === "CloudStorage") {
                                innerHtml += 'Bucket: <a href="HHH" target="_blank">'.replace('HHH', bucketUrl(data.project, data.gcs_options.bucket)) + data.gcs_options.bucket + '</a>';
                            }
                            return innerHtml;
                        }
                    }
                ],
                // selectable: true,
                rowSelectionChanged: function (data, rows) {
                    //rows - array of row components for the selected rows in order of selection
                    //data - array of data objects for the selected rows in order of selection
                    // FIXME: more check for cancel/delete action
                    document.querySelector(".mdc-button.resume_backup").disabled = (0 === data.length);
                    document.querySelector(".mdc-button.cancel_backup").disabled = (0 === data.length);
                    document.querySelector(".mdc-button.delete_backup").disabled = (0 === data.length);
                    this.getRows().forEach(function (row) {
                        if (row.isSelected()) {
                            row.getElement().querySelector('.material-icons').innerHTML = "check_box";
                        } else {
                            row.getElement().querySelector('.material-icons').innerHTML = "check_box_outline_blank";
                        }
                    });
                },
            }
        );

        // refresh backups
        const refreshBackupsButton = mdc.ripple.MDCRipple.attachTo(document.querySelector('.mdc-button.refresh_backup'));
        document.querySelector(".mdc-button.refresh_backup").addEventListener("click", function (event) {
            event.stopPropagation();
            requestListBackups();
        }, {passive: true});

        // backup resume
        const backupResumeDialog = mdc.dialog.MDCDialog.attachTo(document.querySelector('.mdc-dialog.resume_backup'));
        document.querySelector(".mdc-button.resume_backup").addEventListener("click", function (event) {
            event.stopPropagation();
            backupResumeDialog.open();
        }, {passive: true});
        document.querySelector(".mdc-button.resume_backup.abort").addEventListener("click", function (event) {
            event.stopPropagation();
            backupResumeDialog.close();
        }, {passive: true});
        document.querySelector(".mdc-button.resume_backup.confirm").addEventListener("click", function (event) {
            event.stopPropagation();
            backupResumeDialog.close();
            resumeSelectedBackups();
        }, {passive: true});

        function resumeSelectedBackups() {
            let selectedRows = table.getSelectedRows();
            selectedRows.forEach(function (row) {
                openInfo("Backup: resume " + row.getData().id);
                resumeBackup(row.getData().id, function (data) {
                    row.update(data);
                    openInfo("Backup: resumed " + row.getData().id);
                    row.deselect();
                }, function (jqXHR, textStatus, errorThrown) {
                    openInfo("Backup: resume " + errorThrown + ": " + jqXHR.responseText);
                })
            });
        }

        // backup cancel
        const backupCancelDialog = mdc.dialog.MDCDialog.attachTo(document.querySelector('.mdc-dialog.cancel_backup'));
        document.querySelector(".mdc-button.cancel_backup").addEventListener("click", function (event) {
            event.stopPropagation();
            backupCancelDialog.open();
        }, {passive: true});
        document.querySelector(".mdc-button.cancel_backup.abort").addEventListener("click", function (event) {
            event.stopPropagation();
            backupCancelDialog.close();
        }, {passive: true});
        document.querySelector(".mdc-button.cancel_backup.confirm").addEventListener("click", function (event) {
            event.stopPropagation();
            backupCancelDialog.close();
            cancelSelectedBackups();
        }, {passive: true});

        function cancelSelectedBackups() {
            let selectedRows = table.getSelectedRows();
            selectedRows.forEach(function (row) {
                cancelBackup(row.getData().id, function (data) {
                    openInfo("Backup: pausing " + row.getData().id);
                    row.update(data);
                    openInfo("Backup: paused " + row.getData().id);
                    row.deselect();
                }, function (jqXHR, textStatus, errorThrown) {
                    openInfo("Backup: pause failed " + errorThrown + ": " + jqXHR.responseText);
                })
            });
        }

        // backup delete
        const backupDeleteDialog = mdc.dialog.MDCDialog.attachTo(document.querySelector('.mdc-dialog.delete_backup'));
        document.querySelector(".mdc-button.delete_backup").addEventListener("click", function (event) {
            event.stopPropagation();
            backupDeleteDialog.open();
        }, {passive: true});
        document.querySelector(".mdc-button.delete_backup.abort").addEventListener("click", function (event) {
            event.stopPropagation();
            backupDeleteDialog.close();
        }, {passive: true});
        document.querySelector(".mdc-button.delete_backup.confirm").addEventListener("click", function (event) {
            event.stopPropagation();
            backupDeleteDialog.close();
            deleteSelectedBackups();
        }, {passive: true});

        function deleteSelectedBackups() {
            let selectedRows = table.getSelectedRows();
            selectedRows.forEach(function (row) {
                deleteBackup(row.getData().id, function (data) {
                    openInfo("Backup: delete " + row.getData().id);
                    row.update(data);
                    openInfo("Backup: deleted " + row.getData().id);
                    row.deselect();
                }, function (jqXHR, textStatus, errorThrown) {
                    openInfo("Backup: delete " + errorThrown + ": " + jqXHR.responseText);
                })
            });
        }


        // TODO: move to Form CreateBackup
        mdc.ripple.MDCRipple.attachTo(document.querySelector('.mdc-button.create_backup'));
        const createBackupDialog = mdc.dialog.MDCDialog.attachTo(document.querySelector('.mdc-dialog.create_backup'));
        createBackupDialog.scrimClickAction = ""; // disable close of dialog after click outside dialog area
        createBackupDialog.escapeKeyAction = ""; // disable close of dialog after click outside dialog area
        document.querySelector('.mdc-button.create_backup').addEventListener("click", function () {
            createBackupDialog.open();
            createBackupDialog.layout();
        }, {passive: true});
        document.querySelector('#create-backup-content-cancel').addEventListener("click", function () {
            event.stopPropagation();
            createBackupDialog.close();
        }, {passive: true});
        document.querySelector('#create-backup-content-create').addEventListener("click", function () {
            event.stopPropagation();
            let requestBackup = requestBackupFromForm();
            openInfo("Backup: Creating");
            createBackup(requestBackup,
                function (data) { // success
                    createBackupDialog.close();
                    openInfo("Backup: Created");
                    requestListBackups();
                },
                function (jqXHR, textStatus, errorThrown) {
                    openInfo("Backup: create " + errorThrown + ": " + jqXHR.responseText);

                });
        }, {passive: true});

        document.querySelector('#create-backup-content-calculate').addEventListener("click", function () {
            event.stopPropagation();
            let requestBackup = requestBackupFromForm();
            openInfo("Backup: Calculating");
            calculateBackup(requestBackup, calculateSuccess, calculateFailed);
        }, {passive: true});

        const createBackupFormProject = mdc.select.MDCSelect.attachTo(document.querySelector(".mdc-select.create-backup-form-project"));
        createBackupFormProject.listen('MDCSelect:change', (event) => {
            fillDatasets();
            fillBuckets();
        });
        mdc.select.MDCSelectHelperText.attachTo(document.querySelector("#create-backup-form-project-text-helper"));
        const createBackupFormType = mdc.select.MDCSelect.attachTo(document.querySelector(".mdc-select.create-backup-form-type"));
        mdc.select.MDCSelectHelperText.attachTo(document.querySelector("#create-backup-form-type-text-helper"));

        // configure backup
        const configureBackupDialog = mdc.dialog.MDCDialog.attachTo(document.querySelector('.mdc-dialog.update_backup'));
        document.querySelector('#update-backup-content-cancel').addEventListener("click", function () {
            event.stopPropagation();
            configureBackupDialog.close();
        }, {passive: true});
        document.querySelector('#update-backup-content-update').addEventListener("click", function () {
            event.stopPropagation();
            let requestBackup = requestUpdateBackupFromForm();
            openInfo("Backup: Updating");
            configureBackup(requestBackup.backup_id, requestBackup,
                function (data) { // success
                    configureBackupDialog.close();
                    openInfo("Backup: updated");
                },
                function (jqXHR, textStatus, errorThrown) {
                    openInfo("Backup: update " + errorThrown + ": " + jqXHR.responseText);
                });
        }, {passive: true});

        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.update-backup-form-storage-include"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.update-backup-form-storage-exclude"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.update-backup-form-bigquery-table"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.update-backup-form-bigquery-excluded_tables"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.update-backup-form-archive-ttm"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.update-backup-form-mirror-ttl"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.update-backup-form-snapshot-ttl"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#update-backup-storage-include-text-helper"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#update-backup-storage-exclude-text-helper"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#update-backup-form-bigquery-table-helper-text"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#update-backup-form-archive-ttm-text-helper"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#update-backup-form-mirror-ttl-text-helper"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#update-backup-form-snapshot-ttl-text-helper"));

        function fillDatasets() {
            let requestBackup = requestBackupFromForm();
            if (requestBackup.type === "BigQuery" && requestBackup.project) {
                openInfo("Fetching datasets list for project " + requestBackup.project);
                datasetsProject(requestBackup.project,
                    function (data) {
                        if (data.datasets && 0 < data.datasets.length) {
                            document.getElementById("create-backup-form-bigquery-dataset").disabled = false;
                            data.datasets.unshift("");
                            selectOptionsFromList("create-backup-form-bigquery-dataset", data.datasets);
                        } else {
                            document.getElementById("create-backup-form-bigquery-dataset").disabled = true;
                            selectOptionsFromList("create-backup-form-bigquery-dataset", [""])
                        }
                        closeInfo();
                    },
                    function (jqXHR, textStatus, errorThrown) {
                        openInfo("Project datasets: list " + errorThrown + ": " + jqXHR.responseText);
                        selectOptionsFromList("create-backup-form-bigquery-dataset", [""])
                    })
            } else {
                document.getElementById("create-backup-form-bigquery-dataset").disabled = true;
                selectOptionsFromList("create-backup-form-bigquery-dataset", [""])
            }
        }

        function fillBuckets() {
            let requestBackup = requestBackupFromForm();
            if (requestBackup.type === "CloudStorage" && requestBackup.project) {
                openInfo("Fetching buckets list for project " + requestBackup.project);
                bucketsProject(requestBackup.project,
                    function (data) {
                        if (data.buckets && 0 < data.buckets.length) {
                            document.getElementById("create-backup-form-storage-bucket").disabled = false;
                            data.buckets.unshift("");
                            selectOptionsFromList("create-backup-form-storage-bucket", data.buckets);
                        } else {
                            document.getElementById("create-backup-form-storage-bucket").disabled = true;
                            selectOptionsFromList("create-backup-form-storage-bucket", [""])
                        }
                        closeInfo();
                    },
                    function (jqXHR, textStatus, errorThrown) {
                        openInfo("Project datasets: list " + errorThrown + ": " + jqXHR.responseText);
                        selectOptionsFromList("create-backup-form-storage-bucket", [""])
                    })
            } else {
                document.getElementById("create-backup-form-storage-bucket").disabled = true;
                selectOptionsFromList("create-backup-form-storage-bucket", [""])
            }
        }

        createBackupFormType.listen('MDCSelect:change', (event) => {
            if (event.detail.value === "BigQuery") {
                document.querySelectorAll('.bq').forEach(function (element) {
                    element.classList.remove('hidden')
                });
                document.querySelectorAll('.gcs').forEach(function (element) {
                    element.classList.add('hidden')
                });
                document.getElementById('create-backup-content-calculate').disabled = false;
                document.getElementById('calculateChart').innerHTML = '';
                document.getElementById('calculateChart').style = '';
                fillDatasets();
            }
            if (event.detail.value === "CloudStorage") {
                document.querySelectorAll('.bq').forEach(function (element) {
                    element.classList.add('hidden')
                });
                document.querySelectorAll('.gcs').forEach(function (element) {
                    element.classList.remove('hidden')
                });
                document.getElementById('create-backup-content-calculate').disabled = false;
                document.getElementById('calculateChart').innerHTML = '';
                document.getElementById('calculateChart').style = '';
                fillBuckets();
            }
        });
        mdc.select.MDCSelect.attachTo(document.querySelector(".mdc-select.create-backup-form-storage-class"));
        mdc.select.MDCSelectHelperText.attachTo(document.querySelector("#create-backup-form-storage-class-text-helper"));
        mdc.select.MDCSelect.attachTo(document.querySelector(".mdc-select.create-backup-form-storage-region"));
        mdc.select.MDCSelectHelperText.attachTo(document.querySelector("#create-backup-form-storage-region-text-helper"));
        const createBackupFormStrategy = mdc.select.MDCSelect.attachTo(document.querySelector(".mdc-select.create-backup-form-strategy"));

        createBackupFormStrategy.listen('MDCSelect:change', (event) => {
            if (event.detail.value === "Snapshot") {
                document.querySelectorAll('.snapshot').forEach(function (element) {
                    element.classList.remove('hidden')
                });
                document.querySelectorAll('.oneshot, .mirror').forEach(function (element) {
                    element.classList.add('hidden')
                });
            } else if (event.detail.value === "Oneshot") {
                document.querySelectorAll('.oneshot').forEach(function (element) {
                    element.classList.remove('hidden')
                });
                document.querySelectorAll('.mirror, .snapshot').forEach(function (element) {
                    element.classList.add('hidden')
                });
            } else if (event.detail.value === "Mirror") {
                document.querySelectorAll('.mirror').forEach(function (element) {
                    element.classList.remove('hidden')
                });
                document.querySelectorAll('.oneshot, .snapshot').forEach(function (element) {
                    element.classList.add('hidden')
                });
            } else {
                document.querySelectorAll('.snapshot, .mirror, .oneshot').forEach(function (element) {
                    element.classList.add('hidden')
                });

            }
        });
        mdc.select.MDCSelectHelperText.attachTo(document.querySelector("#create-backup-form-strategy-text-helper"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-snapshot-ttl"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#create-backup-form-snapshot-ttl-text-helper"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-oneshot-ttl"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#create-backup-form-oneshot-ttl-text-helper"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-mirror-ttl"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#create-backup-form-mirror-ttl-text-helper"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-archive-ttm"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#create-backup-form-archive-ttm-text-helper"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-snapshot-schedule"));
        mdc.textField.MDCTextFieldHelperText.attachTo(document.querySelector("#create-backup-form-snapshot-schedule-text-helper"));
        mdc.select.MDCSelect.attachTo(document.querySelector(".mdc-select.create-backup-form-bigquery-dataset"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-bigquery-table"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-bigquery-excluded_tables"));
        mdc.select.MDCSelect.attachTo(document.querySelector(".mdc-select.create-backup-form-storage-bucket"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-storage-include"));
        mdc.textField.MDCTextField.attachTo(document.querySelector(".mdc-text-field.create-backup-form-storage-exclude"));

        // Login info
        const accountDialog = mdc.dialog.MDCDialog.attachTo(document.querySelector('.mdc-dialog.account_login'));
        accountDialog.scrimClickAction = ""; // disable close of dialog after click outside dialog area
        accountDialog.escapeKeyAction = ""; // disable close of dialog after click outside dialog area
        this.drawer = mdc.drawer.MDCDrawer.attachTo(document.querySelector('.mdc-drawer'));
        this.topAppBar = mdc.topAppBar.MDCTopAppBar.attachTo(document.getElementById('app-bar'));
        this.topAppBar.listen('MDCTopAppBar:nav', () => {
            this.drawer.open = !this.drawer.open;
        });

        function fillUserLoginInfo() {
            jQuery.get("/api/users/me", function (data) {
                if (0 < data.User.Email.length) {
                    $('#user_me_email').text(data.User.Email);
                }
                this.user = new User(data);
                accountDialog.close();

                selectOptionsFromList("create-backup-form-project", this.user.projectsWithRoles(["owner", "editor"]));
                requestListBackups();
            });
        }

        accountDialog.open();
        $(document).ready(function () {
            fillUserLoginInfo();
        });

        function requestListBackups() {
            openInfo("Backup: Refreshing list");
            let projectId = $("#project-list-project").val();
            listBackups(projectId,
                function (data) {
                    openInfo("Backup: List refreshed");
                    if (null == data.backups) {
                        table.setData([]);
                    } else {
                        let backups = [];
                        data.backups.forEach(function (backup) {
                            backup.sink_url = sinkBucketUrl(backup.sink, backup.project);
                            backups.push(backup);
                        });
                        table.setData(backups);
                    }
                },
                function (jqXHR, textStatus, errorThrown) {
                    openInfo("Backup: list " + errorThrown + ": " + jqXHR.responseText);
                });
        }
    }

    App.prototype.handleRestoreBackup = function (source, backupId, restoreAt) {
        restoreBackup(backupId, restoreAt, function (data) {
            if (0 < data.actions.length) {
                let tr = document.createElement('tr');
                let td = document.createElement('td');
                td.setAttribute('colspan', '6');
                let textarea = document.createElement('textarea');
                textarea.classList.add('mdc-text-field__input');
                textarea.setAttribute('cols', '240');
                textarea.setAttribute('rows', data.actions.length * 2);
                let textareaVal = '';
                data.actions.forEach(function (action) {
                    textareaVal += action.action + '\n';
                });
                textarea.value = textareaVal;
                td.appendChild(textarea);
                tr.appendChild(td);
                insertAfter(tr, source.parentNode.parentNode);
            }
        });
    };

    return App;
})();
