package configs

import (
	"os"

	"github.com/illikainen/orch/src/hclang"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/structor"
)

type Config struct { // revive:disable:line-length-limit
	Body            hcl.Body    `json:"-"                                         hcl:"body,remain"`
	DefaultFileMode os.FileMode `json:"default_file_mode" cty:"default_file_mode" hcl:"default_file_mode,optional" default:"420"` // 0644
	DefaultDirMode  os.FileMode `json:"default_dir_mode"  cty:"default_dir_mode"  hcl:"default_dir_mode,optional"  default:"493"` // 0755
	PrivateKey      string      `json:"private_key"       cty:"private_key"       hcl:"private_key,optional"`
	PublicKeys      []string    `json:"public_keys"       cty:"public_keys"       hcl:"public_keys,optional"`
	Sandbox         string      `json:"sandbox"           cty:"sandbox"           hcl:"sandbox,optional"`
	DryRun          bool        `json:"dry_run"           cty:"dry_run"           hcl:",optional"`
}

func (c *Config) PartialDecode() error {
	return nil
}

func (c *Config) Decode(ctx *hclang.EvalContext) error {
	if c.Body != nil {
		spec, err := hclang.GenerateObjectSpec(c)
		if err != nil {
			return err
		}

		var cur *hcl.EvalContext
		if ctx != nil {
			cur, err = ctx.Build()
			if err != nil {
				return err
			}
		}

		value, diags := hcldec.Decode(c.Body, spec, cur)
		if diags != nil {
			return diags
		}

		err = hclang.FromCtyValue(value, c)
		if err != nil {
			return err
		}
	} else {
		err := structor.Apply(c, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
