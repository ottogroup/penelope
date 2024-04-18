package provider

import (
	"fmt"
	"github.com/ottogroup/penelope/pkg/config"
	"strconv"
	"time"
)

func defaultProviderCacheTTL() (time.Duration, error) {
	ttl := time.Minute * 5
	if config.DefaultProvidersCacheTTLEnv.Exist() {
		atoi, err := strconv.Atoi(config.DefaultProvidersCacheTTLEnv.MustGet())
		if err != nil {
			return 0, fmt.Errorf("can not parse TTL from environment variable %s", config.DefaultProviderBucketEnv)
		}
		ttl = time.Minute * time.Duration(atoi)
	}
	return ttl, nil
}
