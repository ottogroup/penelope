package mock

const (
    // DefaultJWTToken dummy token
    DefaultJWTToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjZmYjA1Zjc0MjM2NmVlNGNmNGJjZjQ5Zjk4NGM0ODdlNDVjOGM4M2QiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiIvcHJvamVjdHMvMTIzNDU2Nzg5L2FwcHMvZ2NwLWJhY2t1cCIsImVtYWlsIjoiaGFucy53dXJzdEBleGFtcGxlLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjogdHJ1ZSwiZXhwIjoxNTQ4ODYwNTg1LCJoZCI6ImV4YW1wbGUuY29tIiwiaWF0IjoxNTQ4ODU5OTg1LCJpc3MiOiJodHRwczovL2Nsb3VkLmdvb2dsZS5jb20vaWFwIiwic3ViIjoiYWNjb3VudHMuZ29vZ2xlLmNvbToxMjM0NTY3ODkxMjM0NTY3ODkxMjMifQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
)

var (
    // RetrieveTokenForbiddenHTTPMock request
    RetrieveTokenForbiddenHTTPMock = NewMockedHTTPRequest("POST", "/token", forbiddenResponse)
    // RetrieveAccessTokenHTTPMock request
    RetrieveAccessTokenHTTPMock = NewMockedHTTPRequest("POST", "/token", accessTokenResponse)
    // OauthHTTPMock request
    OauthHTTPMock = NewMockedHTTPRequest("POST", "/o/oauth2/token", oauthResponse)
    // ImpersonationHTTPMock request
    ImpersonationHTTPMock = NewMockedHTTPRequest("POST", "/v1/projects/-/serviceAccounts/.*:generateAccessToken", impersonateResponse)

    // ObjectsExistsHTTPMock request
    ObjectsExistsHTTPMock = NewMockedHTTPRequest("GET", "/storage/v1/b/.*/o", objectsExistsResponse)
    // SinkNotExistsHTTPMock request
    SinkNotExistsHTTPMock = NewMockedHTTPRequest("GET", "/storage/v1/b", sinkNotexistsResponse)
    // BucketAttrsHTTPMock request
    BucketAttrsHTTPMock = NewMockedHTTPRequest("GET", "/storage/v1/b/.*", bucketAttrsResponse)
    // PatchBucketAttrsHTTPMock request
    PatchBucketAttrsHTTPMock = NewMockedHTTPRequest("PATCH", "/storage/v1/b/.*", patchBucketAttrsResponse)
    // SinkCreatedHTTPpMock request
    SinkCreatedHTTPpMock = NewMockedHTTPRequest("POST", "/storage/v1/b", sinkCreatedResponse)
    // SinkDeletedHTTPMock request
    SinkDeletedHTTPMock = NewMockedHTTPRequest("DELETE", "/storage/v1/b", sinkDeletedResponse)
    // DatasetNotAllowedInfoHTTPMock request
    DatasetNotAllowedInfoHTTPMock = NewMockedHTTPRequest("GET", "/bigquery/v2/projects/.*/datasets/not-allowed-dataset", datasetInfoNotAllowedResponse)
    // DatasetNotFoundInfoHTTPMock request
    DatasetNotFoundInfoHTTPMock = NewMockedHTTPRequest("GET", "/bigquery/v2/projects/.*/datasets/unknown-dataset", datasetInfoNotFoundResponse)
    // DatasetInfoHTTPMock request
    DatasetInfoHTTPMock = NewMockedHTTPRequest("GET", "/bigquery/v2/projects/.*/datasets/.*", datasetInfoResponse)

    // TableNotFoundMock request
    TableNotFoundMock = NewMockedHTTPRequest("GET", "/bigquery/v2/projects/.*/datasets/.*/tables/notExistingTable", tableInfoNotFoundResponse)

    // TableInfoHTTPMock request
    TableInfoHTTPMock = NewMockedHTTPRequest("GET", "/bigquery/v2/projects/.*/datasets/.*/tables/.*", tableInfoResponse)
    // TablePartitionQueryHTTPMock request
    TablePartitionQueryHTTPMock = NewMockedHTTPRequest("GET", "/bigquery/v2/projects/.*/datasets/.*/tables/.*/data", hasTablePartitionsResponse)
    // TablePartitionJobHTTPMock request
    TablePartitionJobHTTPMock = NewMockedHTTPRequest("POST", "/bigquery/v2/projects/.*/jobs", getTablePartitionsJobResponse)
    // TablePartitionResultHTTPMock request
    TablePartitionResultHTTPMock = NewMockedHTTPRequest("GET", "/bigquery/v2/projects/.*/queries", getTablePartitionsQueryResponse)
    // ExtractJobResultOkHTTPMock request
    ExtractJobResultOkHTTPMock = NewMockedHTTPRequest("GET", "/bigquery/v2/projects/.*/jobs/.*", getExtractJobResultOkResponse)
)

const (
    tableInfoNotFoundResponse = `HTTP/2.0 404 Not Found
Content-Length: 182
Content-Type: application/json; charset=UTF-8

{"error":{"code":404,"message":"Requested entity was not found.","errors":[{"message":"Requested entity was not found.","domain":"global","reason":"notFound"}],"status":"NOT_FOUND"}}
`

    forbiddenResponse = `HTTP/2.0 403 Forbidden
Content-Type: application/xml; charset=UTF-8

`

    oauthResponse = `HTTP/1.0 200 OK
Cache-Control: private
Content-Type: application/json; charset=utf-8

{
 "access_token": "ya29.Gl2iBrk2JsjNXWSCDqlZvieDAllS6G8NrMw4qdKYgdzWzwfVoa2_8V1_JpggK05qPBl7jRV7mpWmw6cpPFeONWJ5ZdZJhkNIWNqzTsBIWOfAW_GJ_QKzk3KZEixDrew",
 "expires_in": 3600,
 "scope": "https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/plus.me https://www.googleapis.com/auth/cloud-platform",
 "token_type": "Bearer",
 "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjZmYjA1Zjc0MjM2NmVlNGNmNGJjZjQ5Zjk4NGM0ODdlNDVjOGM4M2QiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiIvcHJvamVjdHMvMTIzNDU2Nzg5L2FwcHMvZ2NwLWJhY2t1cCIsImVtYWlsIjoiaGFucy53dXJzdEBleGFtcGxlLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjogdHJ1ZSwiZXhwIjoxNTQ4ODYwNTg1LCJoZCI6ImV4YW1wbGUuY29tIiwiaWF0IjoxNTQ4ODU5OTg1LCJpc3MiOiJodHRwczovL2Nsb3VkLmdvb2dsZS5jb20vaWFwIiwic3ViIjoiYWNjb3VudHMuZ29vZ2xlLmNvbToxMjM0NTY3ODkxMjM0NTY3ODkxMjMifQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}`
    // SQLPasswordStorageResponse mocked
    SQLPasswordStorageResponse = `HTTP/1.0 200 OK
Content-Length: 18
Content-Type: text/plain; charset=utf-8
Date: Thu, 31 Jan 2019 19:36:40 GMT

backupuserpassword`

    impersonateResponse = `HTTP/2.0 200 OK
Content-Length: 372
Content-Type: application/json; charset=UTF-8

{"accessToken":"ya29.c.EuYBogb3qftWXZoXhltkmJ7tRhjkwnct1Xyut1pr_Zh2O5dItjgsA1-E6vonVuZX0mIdoQujJ66xKJInlxgPy_-Y8u1bPkb-_HjgZVEtXY41LYb7C0loMi2mtRYh4aJByuGelN-TWkySk720dchBy0nHBnRYpx8QNpIcrsAt4ZGqs7zu9upIA09ctNj3lkXizm56sJSUyIOVGn0SIBYK4bUPiovG9e_XtFkVKIXDjfz3EAu3ORt22ZjPg06u-nepyamxSKjcsOaK15s35-RdD5721b6bAx6zq75K2JU8u2VgBIsOa5n69jE","expireTime":"2019-01-31T20:24:42Z"}`

    accessTokenResponse = `HTTP/2.0 200 OK
Content-Length: 189
Content-Type: application/json; charset=utf-8

{"access_token":"ya29.c.ElqiBvJnP0LubsFssEBs0nasXkHj8a1KDvrSw4QowUrXin_H8qNTj3ROaMt7k2m5qYvZQCNDPPZU__m_CVv5CnuKJDBVR8nTq99qUOgC_ORaB6nrLnuii97SrE4","expires_in":3600,"token_type":"Bearer"}`

    datasetInfoResponse = `HTTP/2.0 200 OK
Content-Length: 1264
Content-Type: application/json; charset=UTF-8

{"kind":"bigquery#table","etag":"9TXsXH096pBN3QRyqPwRfw==","id":"local-ability:demo_delete_me_backup_target.bq_tables_storage_statistics","selfLink":"https://www.googleapis.com/bigquery/v2/projects/local-ability/datasets/.*/tables/.*","tableReference":{"projectId":"local-ability","datasetId":"demo_delete_me_backup_target","tableId":"bq_tables_storage_statistics"},"schema":{"fields":[{"name":"project_id","type":"STRING","mode":"NULLABLE"},{"name":"dataset_id","type":"STRING","mode":"NULLABLE"},{"name":"table_id","type":"STRING","mode":"NULLABLE"},{"name":"creation_time","type":"INTEGER","mode":"NULLABLE"},{"name":"last_modified_time","type":"INTEGER","mode":"NULLABLE"},{"name":"row_count","type":"INTEGER","mode":"NULLABLE"},{"name":"active_storage_bytes","type":"INTEGER","mode":"NULLABLE"},{"name":"longterm_storage_bytes","type":"INTEGER","mode":"NULLABLE"},{"name":"active_costs_per_month","type":"FLOAT","mode":"NULLABLE"},{"name":"longterm_costs_per_month","type":"FLOAT","mode":"NULLABLE"}]},"timePartitioning":{"type":"DAY","expirationMs":"31536000000"},"numBytes":"17213912","numLongTermBytes":"0","numRows":"143569","creationTime":"1548844109427","expirationTime":"1549362509427","lastModifiedTime":"1548844109427","type":"TABLE","location":"EU"}`

    tableInfoResponse = `HTTP/2.0 200 OK
Content-Length: 1264
Content-Type: application/json; charset=UTF-8

{"kind":"bigquery#table","etag":"9TXsXH096pBN3QRyqPwRfw==","id":"local-ability:demo_delete_me_backup_target.bq_tables_storage_statistics","selfLink":"https://www.googleapis.com/bigquery/v2/projects/local-ability/datasets/.*/tables/.*","tableReference":{"projectId":"local-ability","datasetId":"demo_delete_me_backup_target","tableId":"bq_tables_storage_statistics"},"schema":{"fields":[{"name":"project_id","type":"STRING","mode":"NULLABLE"},{"name":"dataset_id","type":"STRING","mode":"NULLABLE"},{"name":"table_id","type":"STRING","mode":"NULLABLE"},{"name":"creation_time","type":"INTEGER","mode":"NULLABLE"},{"name":"last_modified_time","type":"INTEGER","mode":"NULLABLE"},{"name":"row_count","type":"INTEGER","mode":"NULLABLE"},{"name":"active_storage_bytes","type":"INTEGER","mode":"NULLABLE"},{"name":"longterm_storage_bytes","type":"INTEGER","mode":"NULLABLE"},{"name":"active_costs_per_month","type":"FLOAT","mode":"NULLABLE"},{"name":"longterm_costs_per_month","type":"FLOAT","mode":"NULLABLE"}]},"timePartitioning":{"type":"DAY","expirationMs":"31536000000"},"numBytes":"17213912","numLongTermBytes":"0","numRows":"143569","creationTime":"1548844109427","expirationTime":"1549362509427","lastModifiedTime":"1548844109427","type":"TABLE","location":"EU"}`

    sinkNotexistsResponse = `HTTP/2.0 200 OK
Content-Length: 26
Content-Type: application/json; charset=UTF-8
X-Guploader-Uploadid: AEnB2UpMl0d5zZ3a3BGxywGmwRNoQ01t2dcoMYM8bz50ZDej9V5zwT7oUA0ncN6b2RSdlZletB5OinAuxCdKn-JvRGO-7lYCSPFVVRcWPKANcIVSjjfyxOI

{"kind":"storage#buckets"}`

    bucketAttrsResponse = `HTTP/2.0 200 OK
Content-Length: 462
Content-Type: application/json; charset=UTF-8
X-Guploader-Uploadid: AEnB2UpMl0d5zZ3a3BGxywGmwRNoQ01t2dcoMYM8bz50ZDej9V5zwT7oUA0ncN6b2RSdlZletB5OinAuxCdKn-JvRGO-7lYCSPFVVRcWPKANcIVSjjfyxOI

{"kind":"storage#bucket","id":"local-kebab-database","selfLink":"https://www.googleapis.com/storage/v1/b/test-bucket","projectNumber":"879716749172","name":"test-bucket","timeCreated":"2019-01-09T12:26:24.435Z","updated":"2020-02-13T19:59:03.859Z","metageneration":"4","iamConfiguration":{"bucketPolicyOnly":{"enabled":false},"uniformBucketLevelAccess":{"enabled":false}},"location":"EUROPE-WEST1","locationType":"region","storageClass":"STANDARD","etag":"CAQ="}`

    patchBucketAttrsResponse = `HTTP/2.0 200 OK
Content-Length: 462
Content-Type: application/json; charset=UTF-8
X-Guploader-Uploadid: AEnB2UpMl0d5zZ3a3BGxywGmwRNoQ01t2dcoMYM8bz50ZDej9V5zwT7oUA0ncN6b2RSdlZletB5OinAuxCdKn-JvRGO-7lYCSPFVVRcWPKANcIVSjjfyxOI

{"kind":"storage#bucket","id":"local-kebab-database","selfLink":"https://www.googleapis.com/storage/v1/b/test-bucket","projectNumber":"879716749172","name":"test-bucket","timeCreated":"2019-01-09T12:26:24.435Z","updated":"2020-02-13T19:59:03.859Z","metageneration":"4","iamConfiguration":{"bucketPolicyOnly":{"enabled":false},"uniformBucketLevelAccess":{"enabled":false}},"location":"EUROPE-WEST1","locationType":"region","storageClass":"STANDARD","etag":"CAQ="}`

    objectsExistsResponse = `HTTP/2.0 200 OK
Content-Length: 2658
Content-Type: application/json; charset=UTF-8
X-Guploader-Uploadid: AEnB2UpMl0d5zZ3a3BGxywGmwRNoQ01t2dcoMYM8bz50ZDej9V5zwT7oUA0ncN6b2RSdlZletB5OinAuxCdKn-JvRGO-7lYCSPFVVRcWPKANcIVSjjfyxOI

{"kind":"storage#objects","items":[{"kind":"storage#object","id":"bkp_bq_3aee0db1-097e-44c6-8943-dd9416b94a62/dataset/demo_delete_me_backup_target/table/gcp_billing_budget_amount_plan/daac5314-0472-4bf4-952c-7c418d4ef4f3-000000000000.avro/1552374241274108","selfLink":"https://www.googleapis.com/storage/v1/b/bkp_bq_3aee0db1-097e-44c6-8943-dd9416b94a62/o/dataset%2Fdemo_delete_me_backup_target%2Ftable%2Fbq_tables_storage_statistics$20190312%2Fdaac5314-0472-4bf4-952c-7c418d4ef4f3-000000000000.avro","name":"dataset/demo_delete_me_backup_target/table/gcp_billing_budget_amount_plan/daac5314-0472-4bf4-952c-7c418d4ef4f3-000000000000.avro","bucket":"bkp_bq_3aee0db1-097e-44c6-8943-dd9416b94a62","generation":"1552374241274108","metageneration":"1","contentType":"application/octet-stream","timeCreated":"2019-03-12T07:04:01.273Z","updated":"2019-03-12T07:04:01.273Z","storageClass":"REGIONAL","timeStorageClassUpdated":"2019-03-12T07:04:01.273Z","size":"715","md5Hash":"1i7VMcykXrhq9IprXxwxQw==","mediaLink":"https://www.googleapis.com/download/storage/v1/b/bkp_bq_3aee0db1-097e-44c6-8943-dd9416b94a62/o/dataset%2Fdemo_delete_me_backup_target%2Ftable%2Fbq_tables_storage_statistics$20190312%2Fdaac5314-0472-4bf4-952c-7c418d4ef4f3-000000000000.avro?generation=1552374241274108&alt=media","crc32c":"llRpMw==","etag":"CPzR1tmE/OACEAE="},{"kind":"storage#object","id":"bkp_bq_3aee0db1-097e-44c6-8943-dd9416b94a62/dataset/demo_delete_me_backup_target/table/bq_tables_storage_statistics$20190312/fd45c523-142e-44f9-a358-e57c8ae77df2-000000000000.avro/1552374042413017","selfLink":"https://www.googleapis.com/storage/v1/b/bkp_bq_3aee0db1-097e-44c6-8943-dd9416b94a62/o/dataset%2Fdemo_delete_me_backup_target%2Ftable%2Fbq_tables_storage_statistics$20190312%2Ffd45c523-142e-44f9-a358-e57c8ae77df2-000000000000.avro","name":"dataset/demo_delete_me_backup_target/table/bq_tables_storage_statistics$20190312/fd45c523-142e-44f9-a358-e57c8ae77df2-000000000000.avro","bucket":"bkp_bq_3aee0db1-097e-44c6-8943-dd9416b94a62","generation":"1552374042413017","metageneration":"1","contentType":"application/octet-stream","timeCreated":"2019-03-12T07:00:42.412Z","updated":"2019-03-12T07:00:42.412Z","storageClass":"REGIONAL","timeStorageClassUpdated":"2019-03-12T07:00:42.412Z","size":"718","md5Hash":"+bnIUGFcWChEja40Es/f6A==","mediaLink":"https://www.googleapis.com/download/storage/v1/b/bkp_bq_3aee0db1-097e-44c6-8943-dd9416b94a62/o/dataset%2Fdemo_delete_me_backup_target%2Ftable%2Fbq_tables_storage_statistics$20190312%2Ffd45c523-142e-44f9-a358-e57c8ae77df2-000000000000.avro?generation=1552374042413017&alt=media","crc32c":"UZEh6Q==","etag":"CNmP7fqD/OACEAE="}]}`

    sinkCreatedResponse = `HTTP/2.0 200 OK
Content-Length: 480
Content-Type: application/json; charset=UTF-8

{"kind":"storage#bucket","id":"bkp_bq_926a9097-7b1e-487b-b4a0-8a6418074fd0","selfLink":"https://www.googleapis.com/storage/v1/b/bkp_bq_926a9097-7b1e-487b-b4a0-8a6418074fd0","projectNumber":"386127437557","name":"bkp_bq_926a9097-7b1e-487b-b4a0-8a6418074fd0","timeCreated":"2019-01-31T20:45:17.838Z","updated":"2019-01-31T20:45:17.838Z","metageneration":"1","iamConfiguration":{"bucketPolicyOnly":{"enabled":false}},"location":"EUROPE-WEST1","storageClass":"STANDARD","etag":"CAE="}`

    sinkDeletedResponse = `HTTP/2.0 204 OK                                                                                                                                     
content-type: application/json                                                                                                                   

`

    hasTablePartitionsResponse = `HTTP/2.0 200 OK
Content-Length: 3283
Content-Type: application/json; charset=UTF-8

{"kind":"bigquery#tableDataList","etag":"K8m8os7SpX+kH8cMEoQFCg==","totalRows":"76","rows":[{"f":[{"v":"1375"},{"v":"1546041600.0"}]},{"f":[{"v":"1424"},{"v":"1546560000.0"}]},{"f":[{"v":"1301"},{"v":"1544918400.0"}]},{"f":[{"v":"1492"},{"v":"1546992000.0"}]},{"f":[{"v":"42521"},{"v":"1542326400.0"}]},{"f":[{"v":"1727"},{"v":"1548547200.0"}]},{"f":[{"v":"1127"},{"v":"1543622400.0"}]},{"f":[{"v":"1292"},{"v":"1544486400.0"}]},{"f":[{"v":"1336"},{"v":"1545696000.0"}]},{"f":[{"v":"1480"},{"v":"1546646400.0"}]},{"f":[{"v":"1015"},{"v":"1542844800.0"}]},{"f":[{"v":"1546"},{"v":"1547078400.0"}]},{"f":[{"v":"1369"},{"v":"1545955200.0"}]},{"f":[{"v":"1378"},{"v":"1545350400.0"}]},{"f":[{"v":"1317"},{"v":"1544832000.0"}]},{"f":[{"v":"1091"},{"v":"1543708800.0"}]},{"f":[{"v":"1040"},{"v":"1542758400.0"}]},{"f":[{"v":"1115"},{"v":"1544140800.0"}]},{"f":[{"v":"1308"},{"v":"1545091200.0"}]},{"f":[{"v":"1717"},{"v":"1548288000.0"}]},{"f":[{"v":"1457"},{"v":"1546819200.0"}]},{"f":[{"v":"982"},{"v":"1542412800.0"}]},{"f":[{"v":"1139"},{"v":"1544313600.0"}]},{"f":[{"v":"1074"},{"v":"1543363200.0"}]},{"f":[{"v":"1110"},{"v":"1544054400.0"}]},{"f":[{"v":"1695"},{"v":"1548201600.0"}]},{"f":[{"v":"1404"},{"v":"1546300800.0"}]},{"f":[{"v":"1237"},{"v":"1544659200.0"}]},{"f":[{"v":"1734"},{"v":"1548633600.0"}]},{"f":[{"v":"1628"},{"v":"1547942400.0"}]},{"f":[{"v":"1101"},{"v":"1543536000.0"}]},{"f":[{"v":"1089"},{"v":"1542931200.0"}]},{"f":[{"v":"1094"},{"v":"1543017600.0"}]},{"f":[{"v":"1446"},{"v":"1546732800.0"}]},{"f":[{"v":"1746"},{"v":"1548720000.0"}]},{"f":[{"v":"1624"},{"v":"1547856000.0"}]},{"f":[{"v":"1145"},{"v":"1544400000.0"}]},{"f":[{"v":"1340"},{"v":"1545177600.0"}]},{"f":[{"v":"1393"},{"v":"1546214400.0"}]},{"f":[{"v":"1045"},{"v":"1543276800.0"}]},{"f":[{"v":"1635"},{"v":"1548028800.0"}]},{"f":[{"v":"1413"},{"v":"1546387200.0"}]},{"f":[{"v":"1089"},{"v":"1543881600.0"}]},{"f":[{"v":"1799"},{"v":"1548806400.0"}]},{"f":[{"v":"1524"},{"v":"1547251200.0"}]},{"f":[{"v":"1506"},{"v":"1546905600.0"}]},{"f":[{"v":"1722"},{"v":"1548374400.0"}]},{"f":[{"v":"1726"},{"v":"1548460800.0"}]},{"f":[{"v":"1281"},{"v":"1545004800.0"}]},{"f":[{"v":"1033"},{"v":"1542672000.0"}]},{"f":[{"v":"1580"},{"v":"1547596800.0"}]},{"f":[{"v":"1059"},{"v":"1543795200.0"}]},{"f":[{"v":"1352"},{"v":"1545264000.0"}]},{"f":[{"v":"1668"},{"v":"1548115200.0"}]},{"f":[{"v":"1049"},{"v":"1543104000.0"}]},{"f":[{"v":"1601"},{"v":"1547769600.0"}]},{"f":[{"v":"1135"},{"v":"1544227200.0"}]},{"f":[{"v":"1530"},{"v":"1547337600.0"}]},{"f":[{"v":"1601"},{"v":"1547683200.0"}]},{"f":[{"v":"1319"},{"v":"1545523200.0"}]},{"f":[{"v":"1094"},{"v":"1543449600.0"}]},{"f":[{"v":"1574"},{"v":"1547510400.0"}]},{"f":[{"v":"1327"},{"v":"1545609600.0"}]},{"f":[{"v":"1384"},{"v":"1546128000.0"}]},{"f":[{"v":"1345"},{"v":"1545782400.0"}]},{"f":[{"v":"1322"},{"v":"1545436800.0"}]},{"f":[{"v":"1517"},{"v":"1547164800.0"}]},{"f":[{"v":"1210"},{"v":"1544572800.0"}]},{"f":[{"v":"1098"},{"v":"1543968000.0"}]},{"f":[{"v":"982"},{"v":"1542499200.0"}]},{"f":[{"v":"1452"},{"v":"1546473600.0"}]},{"f":[{"v":"1024"},{"v":"1543190400.0"}]},{"f":[{"v":"983"},{"v":"1542585600.0"}]},{"f":[{"v":"1287"},{"v":"1544745600.0"}]},{"f":[{"v":"1540"},{"v":"1547424000.0"}]},{"f":[{"v":"1354"},{"v":"1545868800.0"}]}]}`

    getTablePartitionsJobResponse = `HTTP/2.0 200 OK
Content-Length: 1053
Content-Type: application/json; charset=UTF-8

{"kind":"bigquery#job","etag":"\"_gQ-oglsj7Mb0XnOYn45kL7hlkw/DCLtR2K7_K8EsYX3UFAEC3nWczM\"","id":"local-ability-backup:EU.PYRwoNSDhUNuqUtVbxtkjbSvmx7","selfLink":"https://www.googleapis.com/bigquery/v2/projects/local-ability-backup/jobs/PYRwoNSDhUNuqUtVbxtkjbSvmx7?location=EU","jobReference":{"projectId":"local-ability-backup","jobID":"PYRwoNSDhUNuqUtVbxtkjbSvmx7","location":"EU"},"configuration":{"jobType":"QUERY","query":{"query":"SELECT count(*) as total, _PARTITIONTIME as p FROM local-ability.demo_delete_me_backup_target.bq_tables_storage_statistics WHERE _PARTITIONTIME IS NOT NULL GROUP BY p","destinationTable":{"projectId":"local-ability-backup","datasetId":"_c435c96a4f5f664be4c9640bcb0109ed1e31c2ca","tableId":"anon71483bc94f82b140d31ac88bb38697c9dc860ba1"},"createDisposition":"CREATE_IF_NEEDED","writeDisposition":"WRITE_TRUNCATE","priority":"INTERACTIVE","useLegacySql":false}},"status":{"state":"RUNNING"},"statistics":{"creationTime":"1548967518781","startTime":"1548967519178","query":{"statementType":"SELECT"}},"user_email":".*"}`

    getTablePartitionsQueryResponse = `HTTP/2.0 200 OK
Content-Length: 386
Content-Type: application/json; charset=UTF-8

{"kind":"bigquery#getQueryResultsResponse","etag":"P0Ea2Yx1PihRPFrsH/Q3fA==","schema":{"fields":[{"name":"total","type":"INTEGER","mode":"NULLABLE"},{"name":"p","type":"TIMESTAMP","mode":"NULLABLE"}]},"jobReference":{"projectId":"local-ability-backup","jobID":"PYRwoNSDhUNuqUtVbxtkjbSvmx7","location":"EU"},"totalRows":"76","totalBytesProcessed":"0","jobComplete":true,"cacheHit":false}`

    getExtractJobResultOkResponse = `HTTP/2.0 200 OK
Content-Length: 1191
Content-Type: application/json; charset=UTF-8

{"kind":"bigquery#job","id":"local-ability-backup:EU.CmvErmA5U1tPaOyPB3w8ukAkooi","selfLink":"https://www.googleapis.com/bigquery/v2/projects/local-ability-backup/jobs/CmvErmA5U1tPaOyPB3w8ukAkooi?location=EU","user_email":"backup-account@local-ability-backup.iam.gserviceaccount.com","configuration":{"extract":{"sourceTable":{"projectId":"local-ability","datasetId":"demo_delete_me_backup_target","tableId":"amount_budget_plan"},"destinationURI":"gs://bkp_bq_8c2b59b7-a004-4344-bbc5-d625aab328e2/dataset/demo_delete_me_backup_target/table/amount_budget_plan/c194f329-002b-4d7f-902d-8b8e05f886e0-*.avro","destinationURIs":["gs://bkp_bq_8c2b59b7-a004-4344-bbc5-d625aab328e2/dataset/demo_delete_me_backup_target/table/amount_budget_plan/c194f329-002b-4d7f-902d-8b8e05f886e0-*.avro"],"destinationFormat":"AVRO"},"jobType":"EXTRACT"},"jobReference":{"projectId":"local-ability-backup","jobID":"CmvErmA5U1tPaOyPB3w8ukAkooi","location":"EU"},"statistics":{"creationTime":"1548943501165","startTime":"1548943501381","endTime":"1548943502111","extract":{"destinationURIFileCounts":["1"]},"totalSlotMs":"137","reservationUsage":[{"name":"default-pipeline","slotMs":"137"}]},"status":{"state":"DONE"}}`

    datasetInfoNotFoundResponse = `HTTP/2.0 404 Not Found
Content-Length: 182
Content-Type: application/json; charset=UTF-8

{"error":{"code":404,"message":"Requested entity was not found.","errors":[{"message":"Requested entity was not found.","domain":"global","reason":"notFound"}],"status":"NOT_FOUND"}}`

    datasetInfoNotAllowedResponse = `HTTP/2.0 403 Forbidden
Content-Type: application/xml; charset=UTF-8

`
)
