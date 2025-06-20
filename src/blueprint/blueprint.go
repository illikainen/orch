package blueprint

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/bindings"
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/fact"
	"github.com/illikainen/orch/src/hosts"
	"github.com/illikainen/orch/src/includes"
	"github.com/illikainen/orch/src/metadata"
	"github.com/illikainen/orch/src/rpc"
	"github.com/illikainen/orch/src/tasks/outputs"
	"github.com/illikainen/orch/src/utils"
	"github.com/illikainen/orch/src/variables"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/assoc"
	"github.com/illikainen/go-utils/src/base64"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/sandbox"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type Filter struct {
	Hosts []string
	Tags  []string
}

type Options struct {
	Path         string
	Config       *configs.Config
	Filter       Filter
	Sandbox      sandbox.Sandbox
	DryRun       bool
	AllowMissing bool
}

type Blueprint struct {
	Includes     includes.Includes   `hcl:"include,block"`
	Config       *configs.Config     `hcl:"config,block"`
	Variables    variables.Variables `hcl:"var,block"`
	Hosts        hosts.Hosts         `hcl:"host,block"`
	Bindings     bindings.Bindings   `hcl:"bind,block"`
	Dependencies Dependencies
	facts        *fact.Facts
	output       outputs.Outputs
	functions    map[string]function.Function
	opts         *Options
}

func NewBlueprint(opts *Options) *Blueprint {
	return &Blueprint{
		Config:       fn.Ternary(opts.Config != nil, opts.Config, &configs.Config{}),
		Dependencies: map[string][]string{},
		functions:    localFunctions(),
		opts:         opts,
	}
}

func (b *Blueprint) PartialDecode() error {
	err := b.partialDecodeMerge(b.opts.Path)
	if err != nil {
		return err
	}

	err = b.Includes.PartialDecode(filepath.Dir(b.opts.Path))
	if err != nil {
		return err
	}

	for _, include := range b.Includes {
		err := b.partialDecodeMerge(include.Src)
		if err != nil {
			return err
		}
	}

	if b.opts.DryRun {
		b.Config.DryRun = b.opts.DryRun
	}

	err = b.Config.PartialDecode()
	if err != nil {
		return err
	}

	err = b.Variables.PartialDecode()
	if err != nil {
		return err
	}

	err = b.Hosts.PartialDecode(&hosts.Filter{
		Hosts: b.opts.Filter.Hosts,
		Tags:  b.opts.Filter.Tags,
	})
	if err != nil {
		return err
	}

	err = b.Bindings.PartialDecode(filepath.Dir(b.opts.Path))
	if err != nil {
		return err
	}

	for _, host := range b.Hosts {
		deps := host.Dependencies
		for _, binding := range b.Bindings {
			if binding.Match(host) {
				deps = append(deps, binding.Dependencies...)
			}
		}
		deps = seq.Uniq(seq.Filter(deps, host.Name))

		for _, dep := range deps {
			if !seq.ContainsBy(b.Hosts, func(h *hosts.Host) bool {
				return h.Name == dep
			}) {
				return errors.Errorf("%s depends on %s but %s is not scheduled to be applied",
					host.Name, dep, dep)
			}
		}

		b.Dependencies[host.Name] = deps
	}

	return nil
}

func (b *Blueprint) partialDecodeMerge(path string) (err error) {
	stat, err := os.Stat(path)
	if err != nil {
		if b.opts.AllowMissing && os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if stat.IsDir() {
		return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(p))
			if ext == ".hcl" || ext == ".hclseal" {
				err := b.partialDecodeMerge(p)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}

	log.Debugf("decoding %s", path)
	input, err := os.Open(path) // #nosec G304
	if err != nil {
		return err
	}
	defer errorx.Defer(input.Close, &err)

	var data []byte
	if strings.ToLower(filepath.Ext(path)) == ".hclseal" {
		keys, err := blob.ReadKeyring(b.Config.PrivateKey, b.Config.PublicKeys)
		if err != nil {
			return err
		}

		decoder, err := base64.NewDecoder(base64.StdEncoding.Strict(), input)
		if err != nil {
			return err
		}

		blobber, err := blob.NewReader(decoder, &blob.Options{
			Type:      metadata.Name(),
			Keyring:   keys,
			Encrypted: true,
		})
		if err != nil {
			return err
		}

		buf := &bytes.Buffer{}
		_, err = io.Copy(buf, blobber)
		if err != nil {
			return err
		}

		data = buf.Bytes()
	} else {
		buf := &bytes.Buffer{}
		_, err := io.Copy(buf, input)
		if err != nil {
			return err
		}

		data = buf.Bytes()
	}

	hcl := hclparse.NewParser()
	hclFile, diags := hcl.ParseHCL(data, path)
	if diags != nil {
		return diags
	}

	tmp := &Blueprint{}
	diags = gohcl.DecodeBody(hclFile.Body, nil, tmp)
	if diags != nil {
		return diags
	}

	b.Includes = append(b.Includes, tmp.Includes...)
	b.Config = fn.Ternary(tmp.Config != nil, tmp.Config, b.Config)
	b.Variables = append(b.Variables, tmp.Variables...)
	b.Hosts = append(b.Hosts, tmp.Hosts...)
	b.Bindings = append(b.Bindings, tmp.Bindings...)
	return nil
}

func (b *Blueprint) Apply(name string, o outputs.Outputs) (output outputs.Outputs, err error) {
	b.output = o

	err = b.Includes.Decode(b.evalContext)
	if err != nil {
		return nil, err
	}

	err = b.Config.Decode(b.evalContext)
	if err != nil {
		return nil, err
	}

	err = b.Variables.Decode(b.evalContext)
	if err != nil {
		return nil, err
	}

	host, ok := seq.FindBy(b.Hosts, func(h *hosts.Host) bool {
		return h.Name == name
	})
	if !ok {
		return nil, errors.Errorf("invalid host: %s", name)
	}

	err = host.Decode(b.evalContext)
	if err != nil {
		return nil, err
	}

	err = host.Connector.Dial()
	if err != nil {
		return nil, err
	}
	defer errorx.Defer(host.Connector.Close, &err)

	err = host.Connector.UploadBinary()
	if err != nil {
		return nil, err
	}

	ctrl, err := host.Connector.Start()
	if err != nil {
		return nil, err
	}
	defer errorx.Defer(ctrl.Close, &err)

	factsData, err := ctrl.Call(&rpc.FunctionCall{
		Function: "gather_facts",
	})
	if err != nil {
		return nil, err
	}

	var facts fact.Facts
	err = json.Unmarshal(factsData, &facts)
	if err != nil {
		return nil, err
	}
	b.facts = &facts

	b.functions = assoc.Merge(b.functions, host.Connector.Functions())

	for _, binding := range b.Bindings {
		if !binding.Match(host) {
			continue
		}

		err := binding.Decode(b.evalContext)
		if err != nil {
			return nil, err
		}

		for _, role := range binding.Roles {
			for _, task := range role.Tasks {
				err := task.Decode(role.Name, host.Name, b.evalContext, b.Config)
				if err != nil {
					return nil, err
				}

				if !task.Include() {
					continue
				}

				out, err := task.Apply(ctrl)
				if err != nil {
					return nil, errors.Errorf("%s: %s.%s: %s", host.Name, role.Name, task.Name, err)
				}

				b.output = append(b.output, out)
				output = append(output, out)

				status := "up-to-date"
				if out.IsChanged() {
					status = "changed"
				}
				log.Infof("%s: %s.%s: %s", host.Name, role.Name, task.Name, status)
				for typ, diffs := range out.Differences() {
					if len(diffs) > 0 {
						log.Infof("    %s\n    %s\n", typ, strings.Repeat("-", len(typ)))

						for _, diff := range diffs {
							log.Infof("    %s", diff)
						}
						log.Info()
					}
				}
			}
		}
	}

	return output, nil
}

func (b *Blueprint) evalContext() (*hcl.EvalContext, error) {
	ctx := &hcl.EvalContext{
		Functions: b.functions,
		Variables: map[string]cty.Value{},
	}

	facts, err := b.facts.Variables()
	if err != nil {
		return nil, err
	}

	ctx.Variables, err = utils.MergeCtyValues(ctx.Variables, facts)
	if err != nil {
		return nil, err
	}

	ctx.Variables, err = utils.MergeCtyValues(ctx.Variables, b.Variables.Variables())
	if err != nil {
		return nil, err
	}

	ctx.Variables, err = utils.MergeCtyValues(ctx.Variables, b.Hosts.Variables())
	if err != nil {
		return nil, err
	}

	ctx.Variables, err = utils.MergeCtyValues(ctx.Variables, b.Bindings.Variables())
	if err != nil {
		return nil, err
	}

	output, err := b.output.Variables()
	if err != nil {
		return nil, err
	}
	ctx.Variables, err = utils.MergeCtyValues(ctx.Variables, output)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}
