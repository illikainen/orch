package utils

import (
	"bytes"
	"fmt"

	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func FormatDiff(diffs []diffmatchpatch.Diff) (string, error) {
	buf := bytes.Buffer{}

	for _, diff := range diffs {
		prefix := ""
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			prefix = "+"
		case diffmatchpatch.DiffDelete:
			prefix = "-"
		}

		for _, line := range stringx.SplitLines(diff.Text) {
			str := fmt.Sprintf("%s %s\n", prefix, line)
			n, err := buf.WriteString(str)
			if err != nil {
				return "", err
			}
			if n != len(str) {
				return "", errors.Errorf("invalid write size")
			}
		}
	}

	return buf.String(), nil
}
