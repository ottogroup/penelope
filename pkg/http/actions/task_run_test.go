package actions

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"
)

func init() {
	os.Setenv("TASKS_VALIDATION_HTTP_HEADER_NAME", "")
	os.Setenv("TASKS_VALIDATION_HTTP_HEADER_VALUE", "")
	os.Setenv("TASKS_VALIDATION_ALLOWED_IP_ADDRESSES", "")
}

func TestRestoringBackupHandler_validateRequest_NoValidationsDefined(t *testing.T) {
	r := &http.Request{
		Header:     nil,
		RemoteAddr: "",
	}
	err := validateRequest(r)
	assert.NoError(t, err)
}

func TestRestoringBackupHandler_validateRequest_ExpectHeader(t *testing.T) {
	os.Setenv("TASKS_VALIDATION_HTTP_HEADER_NAME", "TASK-HEADER")
	defer os.Setenv("TASKS_VALIDATION_HTTP_HEADER_NAME", "")

	r := &http.Request{
		Header:     map[string][]string{},
		RemoteAddr: "",
	}
	err := validateRequest(r)
	assert.Error(t, err)

	os.Setenv("TASKS_VALIDATION_HTTP_HEADER_VALUE", "VALUE")
	defer os.Setenv("TASKS_VALIDATION_HTTP_HEADER_VALUE", "")

	err = validateRequest(r)
	assert.Error(t, err)

	// Now setting expected value
	r.Header.Set("TASK-HEADER", "VALUE")
	err = validateRequest(r)
	assert.NoError(t, err)

	fmt.Printf("%s \n", r.Header)
}

func TestRestoringBackupHandler_validateRequest_AllowedIP(t *testing.T) {
	os.Setenv("TASKS_VALIDATION_ALLOWED_IP_ADDRESSES", "127.0.0.1")
	defer os.Setenv("TASKS_VALIDATION_ALLOWED_IP_ADDRESSES", "")

	r := &http.Request{
		Header:     map[string][]string{},
		RemoteAddr: "127.0.0.1:8080",
	}
	err := validateRequest(r)
	assert.NoError(t, err)

	r.RemoteAddr = ""
	err = validateRequest(r)
	assert.Error(t, err)

	r.Header.Set("X-REAL-IP", "10.0.0.1")
	err = validateRequest(r)
	assert.Error(t, err)
}

func TestRestoringBackupHandler_validateRequest_Forwarded(t *testing.T) {
	os.Setenv("TASKS_VALIDATION_ALLOWED_IP_ADDRESSES", "127.0.0.1")
	defer os.Setenv("TASKS_VALIDATION_ALLOWED_IP_ADDRESSES", "")

	r := &http.Request{
		Header:     map[string][]string{},
		RemoteAddr: "10.0.0.1:8080",
	}

	r.Header.Set("X-FORWARDED-FOR", "127.0.0.1,10.0.0.1")
	err := validateRequest(r)
	assert.NoError(t, err)

	r.Header.Set("X-FORWARDED-FOR", "80.0.0.1,10.0.0.1")
	err = validateRequest(r)
	assert.Error(t, err)

	r.Header.Set("X-REAL-IP", "10.0.0.1")
	err = validateRequest(r)
	assert.Error(t, err, "should use x real ip header for validation")
}

func TestRestoringBackupHandler_validateRequest_MultipleIPAddresses(t *testing.T) {
	os.Setenv("TASKS_VALIDATION_ALLOWED_IP_ADDRESSES", "10.0.0.1;127.0.0.1")
	defer os.Setenv("TASKS_VALIDATION_ALLOWED_IP_ADDRESSES", "")

	r := &http.Request{
		Header:     map[string][]string{},
		RemoteAddr: "127.0.0.1:8080",
	}

	err := validateRequest(r)
	assert.NoError(t, err)
}
