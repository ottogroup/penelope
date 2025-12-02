package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPascalCaseToSnakeCase(t *testing.T) {
	assert.Equal(t, "cloud_storage", PascalCaseToSnakeCase("CloudStorage"))
	assert.Equal(t, "big_query", PascalCaseToSnakeCase("BigQuery"))
}
