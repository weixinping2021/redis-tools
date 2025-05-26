package analyzer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type SlowLogEntry struct {
	Timestamp int64
	ExecTime  int64
	Command   string
}

func FetchSlowLogs(addr string, password string, topN int) error {

	var nodes []string
	ctx := context.Background()

	// Check if the Redis instance is a cluster
	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{addr},
		Password: password,
	})

	info, err := clusterClient.ClusterInfo(ctx).Result()
	if err == nil && strings.Contains(info, "cluster_state:ok") {
		fmt.Println("Detected a cluster. Collecting all node addresses...")

		// Fetch all nodes in the cluster
		nodesInfo, err := clusterClient.ClusterNodes(ctx).Result()
		if err != nil {
			return fmt.Errorf("failed to fetch cluster nodes: %v", err)
		}

		for _, line := range splitLines(nodesInfo) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				nodes = append(nodes, strings.Split(parts[1], "@")[0]) // Add node address to the list
			}
		}
	} else {
		fmt.Println("Not a cluster. Proceeding with standalone mode...")
		nodes = append(nodes, addr)
	}

	for _, address := range nodes {
		fmt.Println(address)
		opt := &redis.Options{
			Addr:     address,
			Password: password,
		}

		client := redis.NewClient(opt)
		info, err := client.Info(ctx, "replication").Result()
		if err != nil {
			return err
		}

		fmt.Println(info[0])

		res, err := client.Do(ctx, "SLOWLOG", "GET", topN).Result()

		if err != nil {
			return err
		}

		logs, ok := res.([]interface{})
		if !ok {
			return fmt.Errorf("unexpected SLOWLOG GET result")
		}

		for _, entry := range logs {
			logEntry, ok := entry.([]interface{})
			if !ok || len(logEntry) < 3 {
				continue
			}

			timestamp, _ := logEntry[1].(int64)
			duration, _ := logEntry[2].(int64)
			args := logEntry[3]

			readableTime := time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
			durationMs := float64(duration) / 1000.0

			fmt.Printf("Time: %s | Duration: %.2f ms | Command: %v\n", readableTime, durationMs, args)
		}
	}
	return nil
}

func isMaster(info string) bool {
	// 判断角色
	for _, line := range splitLines(info) {
		if len(line) >= 5 && line[:5] == "role:" {
			return line[5:] == "master"
		}
	}
	return false
}
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := range s {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
