package configs

import (
	"os"

	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

type Config struct {
	Body            hcl.Body    `json:"-"                 hcl:"body,remain"`
	DefaultFileMode os.FileMode `json:"default_file_mode" cty:"default_file_mode"`
	DefaultDirMode  os.FileMode `json:"default_dir_mode"  cty:"default_dir_mode"`
	PrivateKey      string      `json:"private_key"       cty:"private_key"`
	PublicKeys      []string    `json:"public_keys"       cty:"public_keys"`
	Sandbox         string      `json:"sandbox"           cty:"sandbox"`
	DryRun          bool        `json:"dry_run"`
	Path            string      `json:"-"`
}

func (c *Config) PartialDecode() error {
	return nil
}

func (c *Config) Decode(ctxfn func() (*hcl.EvalContext, error)) error {
	var ctx *hcl.EvalContext

	if ctxfn != nil {
		var err error
		ctx, err = ctxfn()
		if err != nil {
			return err
		}
	}

	if c.Body != nil {
		value, diags := hcldec.Decode(
			c.Body,
			&hcldec.ObjectSpec{
				"default_file_mode": &hcldec.AttrSpec{
					Name: "default_file_mode",
					Type: cty.Number,
				},
				"default_dir_mode": &hcldec.AttrSpec{
					Name: "default_dir_mode",
					Type: cty.Number,
				},
				"private_key": &hcldec.AttrSpec{
					Name: "private_key",
					Type: cty.String,
				},
				"public_keys": &hcldec.AttrSpec{
					Name: "public_keys",
					Type: cty.List(cty.String),
				},
			},
			ctx,
		)
		if diags != nil {
			return diags
		}

		body, ok := c.Body.(*hclsyntax.Body)
		if !ok {
			return errors.Errorf("invalid body type")
		}
		c.Path = body.SrcRange.Filename

		err := utils.FromCtyValue(value, c)
		if err != nil {
			return err
		}
	}

	if int(c.DefaultFileMode) == 0 {
		c.DefaultFileMode = 0644
	}
	log.Debugf("default file mode: %s", c.DefaultFileMode)

	if int(c.DefaultDirMode) == 0 {
		c.DefaultDirMode = 0755
	}
	log.Debugf("default dir mode: %s", c.DefaultDirMode)

	return nil
}
