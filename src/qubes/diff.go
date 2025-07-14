package qubes

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/slices"
)

type DiffChange struct {
	Name string
	Old  *string
	New  *string
}

type Diff struct {
	path     cmp.Path
	defaults []string
	changes  []DiffChange
}

func (d *Diff) PushStep(ps cmp.PathStep) {
	d.path = append(d.path, ps)
}

func (d *Diff) PopStep() {
	if len(d.path) > 0 {
		d.path = d.path[:len(d.path)-1]
	}
}

func (d *Diff) Report(r cmp.Result) {
	if !r.Equal() {
		tags := tagFromPath(d.path, "mapstructure")
		if len(tags) > 0 {
			orig, next := d.path.Last().Values()

			if next.Kind() == reflect.Ptr && next.IsNil() {
				if !orig.IsNil() {
					hcltags := tagFromPath(d.path, "hcl")
					if len(hcltags) > 0 && slices.Contains(d.defaults, hcltags[0]) {
						d.changes = append(d.changes, DiffChange{
							Name: tags[0],
							Old:  valueToString(orig),
							New:  nil,
						})
					}
				}
			} else {
				d.changes = append(d.changes, DiffChange{
					Name: tags[0],
					Old:  valueToString(orig),
					New:  valueToString(next),
				})
			}
		}
	}
}

func valueToString(v reflect.Value) *string {
	if !v.IsValid() {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	s := fmt.Sprintf("%v", v.Interface())
	return &s
}

func tagFromPath(path cmp.Path, tag string) []string {
	for i := len(path) - 1; i > 0; i-- {
		if step, ok := path[i].(cmp.StructField); ok {
			parent := path[i-1].Type()
			if parent.Kind() == reflect.Ptr {
				parent = parent.Elem()
			}

			if parent.Kind() == reflect.Struct {
				if field, ok := parent.FieldByName(step.Name()); ok {
					t := field.Tag.Get(tag)
					if t != "" {
						elts := strings.Split(t, ",")
						if elts[0] != "-" {
							return elts
						}
					}
				}
			}
			break
		}
	}

	return nil
}
