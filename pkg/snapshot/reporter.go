package snapshot

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
)

type DiffReporter struct {
	path   cmp.Path
	deltas []Delta
}

func (r *DiffReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

func (r *DiffReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

func (r *DiffReporter) Report(rs cmp.Result) {
	if !rs.Equal() {
		vx, vy := r.path.Last().Values()
		d := Delta{path: fmt.Sprintf("%#v", r.path), prev: vx, curr: vy}
		r.deltas = append(r.deltas, d)
	}
}

func (r *DiffReporter) String() string {
	diffs := lo.Map(r.deltas, func(d Delta, _ int) string {
		return d.String()
	})

	return strings.Join(diffs, "\n")
}
