package mock

import (
    "github.com/stretchr/testify/assert"
    "io/ioutil"
    "net/http"
    "testing"
)

func TestHTTPMockHandler_Register_Multiple(t *testing.T)  {
    handler := NewHTTPMockHandler()
    handler.Start()
    defer handler.Stop()

    handler.Register(
        MockedHTTPRequest{
            method:   "GET",
            path:     "/some/path",
            response: `HTTP/1.0 200 OK
Content-Length: 2
Content-Type: text/plain; charset=utf-8
Date: Thu, 31 Jan 2019 19:36:40 GMT

ok`,
        },
        MockedHTTPRequest{
            method:   "POST",
            path:     "/another/one",
            response: `HTTP/1.0 200 OK
Content-Length: 6
Content-Type: text/plain; charset=utf-8
Date: Thu, 31 Jan 2019 19:36:40 GMT

failed`,
        },
        )

    request, err := http.NewRequest("GET", "/some/path", nil)
    assert.NoError(t, err)
    response, err := http.DefaultClient.Do(request)
    assert.NoError(t, err)

    bodyBytes, err := ioutil.ReadAll(response.Body)
    if err != nil {
        assert.NoError(t, err)
    }
    bodyString := string(bodyBytes)
    assert.Equal(t, bodyString, "ok")
}
