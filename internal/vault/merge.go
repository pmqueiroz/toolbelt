package vault

import "sort"

func Merge(local, remote map[string]Entry) (map[string]Entry, []string) {
	merged := make(map[string]Entry, len(remote)+len(local))
	for key, entry := range remote {
		merged[key] = entry
	}
	var upload []string
	for key, entry := range local {
		if _, ok := remote[key]; ok || entry.ID != "" {
			continue
		}
		merged[key] = entry
		upload = append(upload, key)
	}
	sort.Strings(upload)
	return merged, upload
}
