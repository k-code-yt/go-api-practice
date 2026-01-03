package main

import (
	"fmt"
	"strings"
)

func main() {}

func ConcatWithPlus(a, b string) string {
	return a + b
}

func ConcatWithArgsRange(args []string) string {
	res := ""
	for _, s := range args {
		res += s
	}
	return res
}

func ConcatWithBuilder(a, b string) string {
	var builder strings.Builder
	builder.WriteString(a)
	builder.WriteString(b)
	return builder.String()
}

func ConcatWithSprintf(a, b string) string {
	return fmt.Sprintf("%s%s", a, b)
}
