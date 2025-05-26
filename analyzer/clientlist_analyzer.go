package analyzer

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

type ClientStats struct {
	Addr  string
	Count int
}

func AnalyzeClientList(addr, password string) ([]ClientStats, error) {
	ctx := context.Background()
	opt := &redis.Options{
		Addr:     addr,
		Password: password,
	}
	rdb := redis.NewClient(opt)

	res, err := rdb.Do(ctx, "CLIENT", "LIST").Text()
	if err != nil {
		return nil, err
	}

	statsMap := make(map[string]int)
	lines := strings.Split(res, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, " ")
		for _, part := range parts {
			if strings.HasPrefix(part, "addr=") {
				addrPort := strings.TrimPrefix(part, "addr=")
				ip := strings.Split(addrPort, ":")[0]
				statsMap[ip]++
			}
		}
	}

	stats := make([]ClientStats, 0)
	for ip, count := range statsMap {
		stats = append(stats, ClientStats{
			Addr:  ip,
			Count: count,
		})
	}
	return stats, nil
}
