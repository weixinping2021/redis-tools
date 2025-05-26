package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"redis-tools/analyzer"
	"strings"

	"github.com/hdt3213/rdb/helper"
)

type separators []string

func (s *separators) String() string {
	return strings.Join(*s, " ")
}

func (s *separators) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {

	var (
		cmd        = flag.String("c", "", "command for rdb: json")
		rdbPath    = flag.String("rdb", "", "Path to Redis RDB file")
		addr       = flag.String("addr", "127.0.0.1:6379", "Redis server address")
		password   = flag.String("password", "", "Redis password")
		topN       = flag.Int("top", 10, "Top N entries")
		output     = flag.String("o", "", "output file path")
		n          = flag.Int("n", 0, "")
		port       = flag.Int("port", 0, "listen port for web")
		regexExpr  = flag.String("regex", "", "regex expression")
		noExpired  = flag.Bool("no-expired", false, "filter expired keys")
		maxDepth   = flag.Int("max-depth", 0, "max depth of prefix tree")
		concurrent = flag.Int("concurrent", 0, "concurrent number for json converter")
	)
	var seps separators
	var err error
	flag.Var(&seps, "sep", "separator for flame graph")

	flag.Parse()

	var options []interface{}
	if *regexExpr != "" {
		options = append(options, helper.WithRegexOption(*regexExpr))
	}
	if *noExpired {
		options = append(options, helper.WithNoExpiredOption())
	}
	if *concurrent != 0 {
		options = append(options, helper.WithConcurrent(*concurrent))
	}

	var outputFile *os.File
	if *output == "" {
		outputFile = os.Stdout
	} else {
		outputFile, err = os.Create(*output)
		if err != nil {
			fmt.Printf("open output faild: %v", err)
		}
		defer func() {
			_ = outputFile.Close()
		}()
	}

	switch *cmd {
	case "json":
		err = helper.ToJsons(*rdbPath, *output, options...)
	case "memory":
		err = helper.MemoryProfile(*rdbPath, *output, options...)
	case "aof":
		err = helper.ToAOF(*rdbPath, *output, options)
	case "bigkey":
		err = helper.FindBiggestKeys(*rdbPath, *n, outputFile, options...)
	case "prefix":
		err = helper.PrefixAnalyse(*rdbPath, *n, *maxDepth, outputFile, options...)
	case "flamegraph":
		_, err = helper.FlameGraph(*rdbPath, *port, seps, options...)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		<-make(chan struct{})
	case "slowlog":
		err := analyzer.FetchSlowLogs(*addr, *password, *topN)
		if err != nil {
			log.Fatalf("Slowlog fetch failed: %v", err)
		}
	case "clientlist":
		stats, err := analyzer.AnalyzeClientList(*addr, *password)
		if err != nil {
			log.Fatalf("Client list fetch failed: %v", err)
		}
		fmt.Println("Client IP Statistics:")
		for _, s := range stats {
			fmt.Printf("IP: %s | Connections: %d\n", s.Addr, s.Count)
		}
	default:
		flag.Usage()
	}

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
}
