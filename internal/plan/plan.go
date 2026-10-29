package plan

import "context"

type Action struct {
	Description string
	Run         func(ctx context.Context) error
}

func Dedupe(actions []Action) []Action {
	seen := map[string]bool{}
	var unique []Action
	for _, a := range actions {
		if seen[a.Description] {
			continue
		}
		seen[a.Description] = true
		unique = append(unique, a)
	}
	return unique
}
