// Package parse reads files and parses each line
package parse

import (
	"fmt"
	"os"
	"strings"
)

func ReadFiles(files []string) {
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Println(err)
		}

		content := strings.TrimSpace(string(data))

		for line := range strings.SplitSeq(content, "\n") {
			parseLine(line)
		}
	}
}
