package vault

import (
	"reflect"
	"testing"
)

func TestMergeLetsRemoteWin(t *testing.T) {
	local := map[string]Entry{"A": {Value: "local", ID: "1"}}
	remote := map[string]Entry{"A": {Value: "remote", ID: "1"}}
	merged, upload := Merge(local, remote)
	if merged["A"].Value != "remote" || len(upload) != 0 {
		t.Fatalf("merged = %+v, upload = %v", merged, upload)
	}
}

func TestMergeUploadsLocalOnlyKeys(t *testing.T) {
	local := map[string]Entry{"B": {Value: "b"}, "A": {Value: "a"}}
	merged, upload := Merge(local, map[string]Entry{})
	if !reflect.DeepEqual(upload, []string{"A", "B"}) || merged["A"].Value != "a" {
		t.Fatalf("merged = %+v, upload = %v", merged, upload)
	}
}

func TestMergeDropsKeysDeletedRemotely(t *testing.T) {
	local := map[string]Entry{"GONE": {Value: "x", ID: "was-synced"}}
	merged, upload := Merge(local, map[string]Entry{})
	if _, ok := merged["GONE"]; ok || len(upload) != 0 {
		t.Fatalf("merged = %+v, upload = %v", merged, upload)
	}
}
