package tag

import (
	"regexp"
	"strings"

	"github.com/samber/lo"
)

// logging labels
const (
	LoggerName             = "sleeve"
	ControllerOperationKey = "sleeve:controller-operation"
	ObjectVersionKey       = "sleeve:object-version"
)

var logTypes = []string{ControllerOperationKey, ObjectVersionKey}
var pattern = regexp.MustCompile(`{"LogType": "(?:` + strings.Join(logTypes, "|") + `)"}`)

func StripLogKey(line string) string {
	return pattern.ReplaceAllString(line, "")
}

func StripLogKeyFromLines(lines []string) []string {
	return lo.Map(lines, func(line string, _ int) string {
		return StripLogKey(line)
	})
}
