package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/samber/lo"
	"github.com/tgoodwin/sleeve/pkg/snapshot"
)

var inFile = flag.String("logfile", "default.log", "path to the log file")

const (
	ControllerOperation string = "sleeve:controller-operation"
	ObjectVersion       string = "sleeve:object-version"
)

var logTypes = []string{ControllerOperation, ObjectVersion}
var pattern = regexp.MustCompile(`{"LogType": "(?:` + strings.Join(logTypes, "|") + `)"}`)

func stripLogtype(line string) string {
	return pattern.ReplaceAllString(line, "")
}

func stripLogtypeFromLines(lines []string) []string {
	return lo.Map(lines, func(line string, _ int) string {
		return stripLogtype(line)
	})
}

func main() {
	flag.Parse()
	f, err := os.Open(*inFile)
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(err.Error())
	}

	lines = lo.FilterMap(lines, func(l string, _ int) (string, bool) {
		isSleeveLine := strings.Contains(l, "sleeveless")
		parts := strings.SplitN(l, "sleeveless", 2)
		if len(parts) > 1 {
			l = strings.TrimSpace(parts[1])
		}
		return l, isSleeveLine
	})
	// fmt.Println("Sleeveless lines:", lines[0])

	// filter controller ops
	controllerOps := lo.FilterMap(lines, func(l string, _ int) (string, bool) {
		return l, strings.Contains(l, ControllerOperation)
	})
	controllerOps = stripLogtypeFromLines(controllerOps)
	fmt.Println("Controller Operations:", len(controllerOps))

	versions := lo.FilterMap(lines, func(l string, _ int) (string, bool) {
		return l, strings.Contains(l, ObjectVersion)
	})
	versions = stripLogtypeFromLines(versions)
	records := lo.Map(versions, func(v string, i int) snapshot.Record {
		r, err := snapshot.LoadFromString(v)
		if err != nil {
			panic(err.Error())
		}
		return r
	})

	grouped := snapshot.GroupByID(records)
	for k, v := range grouped {
		versionCount := len(v)
		fmt.Printf("Kind: %s, Object ID: %s, versions seen: %d\n", v[0].Kind, k, versionCount)
		for i, r := range v {
			if i < versionCount-1 {
				diff, err := r.Diff(v[i+1])
				if err != nil {
					fmt.Printf("Error diffing: %s\n", err.Error())
				}
				fmt.Printf("%s\n", diff)
			}
		}
	}
}
