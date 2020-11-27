package rest

import (
    "encoding/json"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/http/mock"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/secret"
    "io/ioutil"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestCreateEmptyRequest(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    var body interface{}
    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRequestMissingProject(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     repository.BigQuery.String(),
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRequestMissingStrategy(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Project: "local-project",
        Type:    repository.BigQuery.String(),
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRequestUnknownStrategy(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "unkown",
        Project:  "local-project",
        Type:     repository.BigQuery.String(),
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRequestMissingType(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Project:  "local-project",
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRequestUnknownType(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     "unkown",
        Project:  "local-project",
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRequestMissingRegion(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy:      "snapshot",
        Type:          "bigquery",
        Project:       "local-project",
        TargetOptions: requestobjects.TargetOptions{},
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRequestInvalidRegion(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     "bigquery",
        Project:  "local-project",
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west2", // London
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
func TestCreateRequestInvalidStorageClass(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     "bigquery",
        Project:  "local-project",
        TargetOptions: requestobjects.TargetOptions{
            Region:       "europe-west1",
            StorageClass: "MULTI-REGIONAL",
        },
        BigQueryOptions: requestobjects.BigQueryOptions{
            Dataset: "unknown-dataset",
            Table:   []string{"bq_tables_storage_statistics"},
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateBigQueryRequestMissingDataset(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     "bigquery",
        Project:  "local-project",
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func buildBackupRequestPath() string {
    return "/api/backups"
}

func post(t *testing.T, s *httptest.Server, path string, requestBody interface{}) (*http.Response, []byte) {
    jsonReqBody, err := json.Marshal(requestBody)
    require.NoError(t, err)

    req, err := http.NewRequest("POST", s.URL+path, strings.NewReader(string(jsonReqBody)))
    require.NoError(t, err)

    req.Header.Set(tokenHeaderKey, mock.DefaultJWTToken)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)

    body, err := ioutil.ReadAll(resp.Body)
    require.NoError(t, err)

    return resp, body
}

func get(t *testing.T, s *httptest.Server, path string) (*http.Response, string) {
    req, err := http.NewRequest("GET", s.URL+path, nil)
    require.NoError(t, err)

    req.Header.Set(tokenHeaderKey, mock.DefaultJWTToken)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)

    body, err := ioutil.ReadAll(resp.Body)
    require.NoError(t, err)

    return resp, string(body)
}

func patch(t *testing.T, s *httptest.Server, path string, requestBody interface{}) (*http.Response, string) {
    jsonReqBody, err := json.Marshal(requestBody)
    require.NoError(t, err)

    req, err := http.NewRequest("PATCH", s.URL+path, strings.NewReader(string(jsonReqBody)))
    require.NoError(t, err)

    req.Header.Set(tokenHeaderKey, mock.DefaultJWTToken)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)

    body, err := ioutil.ReadAll(resp.Body)
    require.NoError(t, err)

    return resp, string(body)
}
