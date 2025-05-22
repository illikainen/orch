package roles

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/tasks"
	"github.com/illikainen/orch/src/variables"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

type Role struct {
	Name         string
	Dir          string
	RelativeDir  string
	Tasks        tasks.Tasks         `hcl:"task,block"`
	Variables    variables.Variables `hcl:"var,block"`
	Dependencies []string
}

func (r *Role) PartialDecode() error {
	err := filepath.WalkDir(r.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.ToLower(filepath.Ext(path)) == ".hcl" {
			log.Debugf("decoding %s", path)

			hcl := hclparse.NewParser()
			hclFile, diags := hcl.ParseHCLFile(path)
			if diags != nil {
				return diags
			}

			role := Role{}
			diags = gohcl.DecodeBody(hclFile.Body, nil, &role)
			if diags != nil {
				return diags
			}

			r.Variables = append(r.Variables, role.Variables...)
			r.Tasks = append(r.Tasks, role.Tasks...)
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = r.Variables.PartialDecode()
	if err != nil {
		return err
	}

	err = r.Tasks.PartialDecode()
	if err != nil {
		return err
	}

	r.Dependencies = append(r.Variables.Dependencies(), r.Tasks.Dependencies()...)

	return r.Validate()
}

func (r *Role) Decode(ctxfn func() (*hcl.EvalContext, error)) error {
	err := r.Variables.Decode(ctxfn)
	if err != nil {
		return err
	}

	return nil
}

func (r *Role) Value() cty.Value {
	value := map[string]cty.Value{}
	for _, task := range r.Tasks {
		value[task.Name] = task.Value()
	}
	for _, v := range r.Variables {
		value[v.Name] = v.Value()
	}
	return cty.ObjectVal(value)
}

func (r *Role) Unique() string {
	return r.Name
}

func (r *Role) Validate() error {
	seen := []string{"condition", "task", "name"}

	for _, task := range r.Tasks {
		err := task.Validate()
		if err != nil {
			return err
		}

		if lo.Contains(seen, task.Unique()) {
			return errors.Errorf("task \"%s\" is not unique", task.Unique())
		}
		seen = append(seen, task.Unique())
	}

	for _, v := range r.Variables {
		err := v.Validate()
		if err != nil {
			return err
		}

		if lo.Contains(seen, v.Unique()) {
			return errors.Errorf("task \"%s\" is not unique", v.Unique())
		}
		seen = append(seen, v.Unique())
	}

	return nil
}
