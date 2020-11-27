function selectOptionsFromList(target, options) {
    if (!target || !options) {
        return;
    }
    var select = document.getElementById(target);
    select.innerHTML = "";
    for (i = 0; i < options.length; i++) {
        var htmlOption = document.createElement('option');
        htmlOption.value = options[i];
        htmlOption.innerHTML = options[i];
        select.appendChild(htmlOption);
    }
}

function sinkBucketUrl(project, bucket) {
    var url = "https://console.cloud.google.com/storage/browser/BBB?project=PPP";
    return url.replace('BBB', bucket).replace('PPP', project);
}

function datasetUrl(project, dataset) {
    var url = "https://console.cloud.google.com/bigquery?project=PPP&p=PPP&d=DDD&page=dataset";
    return url.replace(/PPP/gi, project).replace('DDD', dataset);
}

function bucketUrl(project, bucket) {
    var url = "https://console.cloud.google.com/storage/browser/DDD?project=PPP";
    return url.replace('PPP', project).replace('DDD', bucket);
}

function tableUrl(project, dataset, table) {
    var url = "https://console.cloud.google.com/bigquery?project=PPP&p=PPP&d=DDD&t=TTT&page=table";
    return url.replace(/PPP/gi, project).replace(/DDD/gi, dataset).replace(/TTT/gi, table);
}

function simpleDataTime(date) {
    if (!date) {
        return "";
    }
    return moment.utc(date).format("YYYY-MM-DD HH:mm");
}

function insertAfter(newNode, referenceNode) {
    referenceNode.parentNode.insertBefore(newNode, referenceNode.nextSibling);
}
