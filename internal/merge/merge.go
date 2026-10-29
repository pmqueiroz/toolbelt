package merge

import (
	"bytes"
	"slices"
)

const (
	MarkerLocal    = "<<<<<<< local"
	MarkerSplit    = "======="
	MarkerUpstream = ">>>>>>> toolbelt"
)

type Result struct {
	Content   []byte
	Conflicts int
}

func Three(base, local, upstream []byte) Result {
	o, a, b := lines(base), lines(local), lines(upstream)
	toA, toB := match(o, a), match(o, b)

	var out [][]byte
	conflicts := 0
	i, j, k := 0, 0, 0
	for {
		for i < len(o) && toA[i] == j && toB[i] == k {
			out = append(out, o[i])
			i, j, k = i+1, j+1, k+1
		}
		if i == len(o) && j == len(a) && k == len(b) {
			break
		}

		next := i
		for next < len(o) && (toA[next] < 0 || toB[next] < 0) {
			next++
		}
		nextA, nextB := len(a), len(b)
		if next < len(o) {
			nextA, nextB = toA[next], toB[next]
		}

		baseHunk, localHunk, upstreamHunk := o[i:next], a[j:nextA], b[k:nextB]
		switch {
		case equal(localHunk, upstreamHunk), equal(upstreamHunk, baseHunk):
			out = append(out, localHunk...)
		case equal(localHunk, baseHunk):
			out = append(out, upstreamHunk...)
		default:
			conflicts++
			out = append(out, conflict(localHunk, upstreamHunk)...)
		}
		i, j, k = next, nextA, nextB
	}
	return Result{Content: bytes.Join(out, nil), Conflicts: conflicts}
}

func HasMarkers(content []byte) bool {
	for _, line := range lines(content) {
		trimmed := bytes.TrimRight(line, "\r\n")
		if string(trimmed) == MarkerLocal || string(trimmed) == MarkerUpstream {
			return true
		}
	}
	return false
}

func conflict(local, upstream [][]byte) [][]byte {
	out := [][]byte{[]byte(MarkerLocal + "\n")}
	out = append(out, terminated(local)...)
	out = append(out, []byte(MarkerSplit+"\n"))
	out = append(out, terminated(upstream)...)
	return append(out, []byte(MarkerUpstream+"\n"))
}

func terminated(hunk [][]byte) [][]byte {
	if len(hunk) == 0 || bytes.HasSuffix(hunk[len(hunk)-1], []byte("\n")) {
		return hunk
	}
	out := slices.Clone(hunk)
	out[len(out)-1] = append(slices.Clone(out[len(out)-1]), '\n')
	return out
}

func lines(content []byte) [][]byte {
	if len(content) == 0 {
		return nil
	}
	out := bytes.SplitAfter(content, []byte("\n"))
	if len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

func match(base, other [][]byte) []int {
	n, m := len(base), len(other)
	lcs := make([][]int, n+1)
	for x := range lcs {
		lcs[x] = make([]int, m+1)
	}
	for x := n - 1; x >= 0; x-- {
		for y := m - 1; y >= 0; y-- {
			if bytes.Equal(base[x], other[y]) {
				lcs[x][y] = lcs[x+1][y+1] + 1
			} else {
				lcs[x][y] = max(lcs[x+1][y], lcs[x][y+1])
			}
		}
	}

	matched := make([]int, n)
	for x := range matched {
		matched[x] = -1
	}
	x, y := 0, 0
	for x < n && y < m {
		switch {
		case bytes.Equal(base[x], other[y]):
			matched[x] = y
			x, y = x+1, y+1
		case lcs[x+1][y] >= lcs[x][y+1]:
			x++
		default:
			y++
		}
	}
	return matched
}

func equal(a, b [][]byte) bool {
	return slices.EqualFunc(a, b, bytes.Equal)
}
