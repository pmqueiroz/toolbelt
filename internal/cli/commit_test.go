package cli

import (
	"reflect"
	"testing"
)

func TestSplitCommitArgs(t *testing.T) {
	options, message := splitCommitArgs([]string{"--no-verify", "fix", "the", "thing", "-S"})
	if !reflect.DeepEqual(options, []string{"--no-verify", "-S"}) {
		t.Errorf("options = %v", options)
	}
	if message != "fix the thing" {
		t.Errorf("message = %q", message)
	}

	options, message = splitCommitArgs(nil)
	if options != nil || message != "" {
		t.Errorf("empty args gave %v %q", options, message)
	}
}
