package gcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchLabels(t *testing.T) {
	want := map[string]string{
		"env":   "prod",
		"owner": "team-a",
	}
	have := map[string]string{
		"env":   "prod",
		"owner": "team-a",
	}
	assert.True(t, labelsEqual(want, have))
	have["owner"] = "team-b"
	assert.False(t, labelsEqual(want, have))
}
