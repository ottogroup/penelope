package logging

import (
    "github.com/stretchr/testify/assert"
    "testing"
    "time"
)

func TestRequestPerMinuteQuota_IsReached(t *testing.T) {
    const period5seconds = 5
    const limit = 10
    quota := newLoggingQuota(period5seconds, limit)
    for i := 0; i < limit-1; i++ {
        quota.Increment()
    }
    assert.Falsef(t, quota.IsReached(), "quota should not be reached")
    quota.Increment()
    assert.Truef(t, quota.IsReached(), "quota should be reached")
}

func TestRequestPerMinuteQuota_WaitUntilNextPeriod(t *testing.T) {
    const period2seconds = 2
    const limit = 10
    start := time.Now()
    quota := newLoggingQuota(period2seconds, limit)
    for i := 0; i < limit; i++ {
        quota.Increment()
    }
    quota.WaitUntilNextPeriod()
    stop := time.Now()
    diff := stop.Sub(start)
    assert.Truef(t, diff >= period2seconds, "expected minimum time to be %d [s]", period2seconds)
    assert.Truef(t, diff >= period2seconds+1, "expected maximum time to be %d [s]", period2seconds+1)
}

func TestRequestPerMinuteQuota_QuotaEnd(t *testing.T) {
    const period2seconds = 60
    const limit = 10
    quota := newLoggingQuota(period2seconds, limit)
    println(time.Now().String())
    println(quota.quotaEnd.String())
    quota.reset()
    println(time.Now().String())
    println(quota.quotaEnd.String())
    assert.True(t, time.Now().Add(time.Duration(period2seconds+1)*time.Second).After(quota.quotaEnd))
}

func TestBucketPathsGeneration(t *testing.T) {
    expect := "protoPayload.resourceName:\"projects/_/buckets/test/objects/t1\""
    paths := prepareBucketPaths("test", []string{"t1"})
    assert.Equal(t, expect, paths)

    expect = "protoPayload.resourceName:\"projects/_/buckets/test/objects/t1\" OR protoPayload.resourceName:\"projects/_/buckets/test/objects/t2\""
    paths = prepareBucketPaths("test", []string{"t1", "t2"})
    assert.Equal(t, expect, paths)
}
