package nftables

import (
	"bytes"
	_ "embed"
	"net"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/file_manage"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/process"
	"github.com/pkg/errors"
)

func init() {
	fn.Must(worker.Register("nftables", NewExecutor))
}

type Executor struct {
	Tasks  []Task          `json:"tasks"`
	Config *configs.Config `json:"config" hcl:"-"`
}

type Context struct {
	Policy           *Policy
	IngressRules     []string
	EgressRules      []string
	ForwardRules     []string
	PostroutingRules []string
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

//go:embed nftables.conf
var baseRules string

func (e *Executor) Execute() (output any, err error) {
	tmp, tmpCleanup, err := iofs.MkdirTemp()
	if err != nil {
		return nil, err
	}
	defer errorx.Defer(tmpCleanup, &err)

	var changes []string
	for i := range e.Tasks {
		ctx := &Context{Policy: &e.Tasks[i].Policy}
		err := buildIngressRules(e.Tasks[i].Ingress, ctx)
		if err != nil {
			return nil, err
		}

		err = buildEgressRules(e.Tasks[i].Egress, ctx)
		if err != nil {
			return nil, err
		}

		err = buildNATRules(e.Tasks[i].NAT, ctx)
		if err != nil {
			return nil, err
		}

		tpl, err := template.New("nft").Parse(baseRules)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		var out bytes.Buffer
		err = tpl.Execute(&out, ctx)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		tmpFile := filepath.Join(tmp, "nftables.conf")
		err = iofs.WriteFile(tmpFile, bytes.NewReader(out.Bytes()))
		if err != nil {
			return nil, err
		}

		_, err = process.Exec(&process.ExecOptions{
			Command: []string{"nft", "--check", "--file", tmpFile},
		})
		if err != nil {
			return nil, err
		}

		file := e.Tasks[i].File
		if file == "" {
			file = "/etc/sysconfig/nftables.conf"
			exists, err := iofs.Exists(file)
			if err != nil {
				return nil, err
			}
			if !exists {
				file = "/etc/nftables.conf"
			}
		}

		updates, err := file_manage.WriteFile(
			file,
			out.Bytes(),
			e.Config.DefaultFileMode,
			e.Config.DryRun,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, updates...)
	}

	return &outputs.Output{
		Status: fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged),
		Diff: map[string][]string{
			"rules": changes,
		},
	}, nil
}

func buildIngressRules(rules []Ingress, ctx *Context) error {
	for _, rule := range rules {
		ingress := []string{}
		egress := []string{}

		if rule.SrcIface != "" {
			ingress = append(ingress, "iifname", rule.SrcIface)
			egress = append(egress, "oifname", rule.SrcIface)
		}

		if rule.DstIface != "" {
			ingress = append(ingress, "oifname", rule.DstIface)
			egress = append(egress, "iifname", rule.DstIface)
		}

		if rule.SrcAddr != "" {
			ipv4, err := isIPv4(rule.SrcAddr)
			if err != nil {
				return err
			}

			if ipv4 {
				ingress = append(ingress, "ip saddr", rule.SrcAddr)
				egress = append(egress, "ip daddr", rule.SrcAddr)
			} else {
				ingress = append(ingress, "ip6 saddr", rule.SrcAddr)
				egress = append(egress, "ip6 daddr", rule.SrcAddr)
			}
		}

		if rule.DstAddr != "" {
			ipv4, err := isIPv4(rule.DstAddr)
			if err != nil {
				return err
			}

			if ipv4 {
				ingress = append(ingress, "ip daddr", rule.DstAddr)
				egress = append(egress, "ip saddr", rule.DstAddr)
			} else {
				ingress = append(ingress, "ip6 daddr", rule.DstAddr)
				egress = append(egress, "ip6 saddr", rule.DstAddr)
			}
		}

		if rule.Proto == "icmp" {
			ingress = append(ingress, rule.Proto)
			egress = append(egress, rule.Proto)

			if rule.ICMPType != "" {
				ingress = append(ingress, "type", rule.ICMPType)

				typ, err := oppositeICMPType(rule.ICMPType)
				if err != nil {
					return err
				}
				egress = append(egress, "type", typ)
			}
		} else {
			if rule.SrcPort != "" {
				ingress = append(ingress, rule.Proto, "sport", string(rule.SrcPort))
				egress = append(egress, rule.Proto, "dport", string(rule.SrcPort))
			}

			ingress = append(ingress, rule.Proto)
			egress = append(egress, rule.Proto)
			if rule.DstPort != "" {
				ingress = append(ingress, "dport", string(rule.DstPort))
				egress = append(egress, "sport", string(rule.DstPort))
			}
		}

		ingress = append(ingress, "ct state new,established", rule.Action)
		egress = append(egress, "ct state established", rule.Action)

		ctx.IngressRules = append(ctx.IngressRules, strings.Join(ingress, " "))
		ctx.EgressRules = append(ctx.EgressRules, strings.Join(egress, " "))
	}

	return nil
}

func buildEgressRules(rules []Egress, ctx *Context) error {
	for _, rule := range rules {
		ingress := []string{}
		egress := []string{}

		if rule.SrcIface != "" {
			ingress = append(ingress, "iifname", rule.SrcIface)
			egress = append(egress, "oifname", rule.SrcIface)
		}

		if rule.DstIface != "" {
			ingress = append(ingress, "oifname", rule.DstIface)
			egress = append(egress, "iifname", rule.DstIface)
		}

		if rule.SrcAddr != "" {
			ipv4, err := isIPv4(rule.SrcAddr)
			if err != nil {
				return err
			}

			if ipv4 {
				ingress = append(ingress, "ip daddr", rule.SrcAddr)
				egress = append(egress, "ip saddr", rule.SrcAddr)
			} else {
				ingress = append(ingress, "ip6 daddr", rule.SrcAddr)
				egress = append(egress, "ip6 saddr", rule.SrcAddr)
			}
		}

		if rule.DstAddr != "" {
			ipv4, err := isIPv4(rule.DstAddr)
			if err != nil {
				return err
			}

			if ipv4 {
				ingress = append(ingress, "ip saddr", rule.DstAddr)
				egress = append(egress, "ip daddr", rule.DstAddr)
			} else {
				ingress = append(ingress, "ip6 saddr", rule.DstAddr)
				egress = append(egress, "ip6 daddr", rule.DstAddr)
			}
		}

		ingress = append(ingress, "meta", "l4proto", rule.Proto)
		egress = append(egress, "meta", "l4proto", rule.Proto)

		if rule.Proto == "icmp" {
			if rule.ICMPType != "" {
				typ, err := oppositeICMPType(rule.ICMPType)
				if err != nil {
					return err
				}
				ingress = append(ingress, rule.Proto, "type", typ)
				egress = append(egress, rule.Proto, "type", rule.ICMPType)
			}
		} else {
			if rule.SrcPort != "" {
				ingress = append(ingress, rule.Proto, "dport", string(rule.SrcPort))
				egress = append(egress, rule.Proto, "sport", string(rule.SrcPort))
			}

			if rule.DstPort != "" {
				ingress = append(ingress, rule.Proto, "sport", string(rule.DstPort))
				egress = append(egress, rule.Proto, "dport", string(rule.DstPort))
			}
		}

		estEgress := append([]string{}, egress...)
		ingress = append(ingress, "ct state established")
		egress = append(egress, "ct state new")
		estEgress = append(estEgress, "ct state established")

		if rule.User != "" {
			egress = append(egress, "skuid", rule.User)
		}

		if rule.Group != "" {
			egress = append(egress, "skgid", rule.Group)
		}

		ingress = append(ingress, rule.Action)
		egress = append(egress, rule.Action)
		estEgress = append(estEgress, rule.Action)

		ctx.IngressRules = append(ctx.IngressRules, strings.Join(ingress, " "))
		ctx.EgressRules = append(ctx.EgressRules, strings.Join(egress, " "))
		ctx.EgressRules = append(ctx.EgressRules, strings.Join(estEgress, " "))
	}

	return nil
}

func buildNATRules(rules []NAT, ctx *Context) error {
	for _, rule := range rules {
		postrouting := []string{}
		out := []string{}
		in := []string{}

		if rule.SrcIface != "" {
			out = append(out, "iifname", rule.SrcIface)
			in = append(in, "iifname", rule.SrcIface)
		}

		if rule.DstIface != "" {
			postrouting = append(postrouting, "oifname", rule.DstIface)
			out = append(out, "oifname", rule.DstIface)
			in = append(in, "oifname", rule.DstIface)
		}

		if rule.SrcAddr != "" {
			ipv4, err := isIPv4(rule.SrcAddr)
			if err != nil {
				return err
			}

			if ipv4 {
				postrouting = append(postrouting, "ip saddr", rule.SrcAddr)
				out = append(out, "ip saddr", rule.SrcAddr)
				in = append(in, "ip daddr", rule.SrcAddr)
			} else {
				postrouting = append(postrouting, "ip6 saddr", rule.SrcAddr)
				out = append(out, "ip6 saddr", rule.SrcAddr)
				in = append(in, "ip6 daddr", rule.SrcAddr)
			}
		}

		if rule.DstAddr != "" {
			ipv4, err := isIPv4(rule.DstAddr)
			if err != nil {
				return err
			}

			if ipv4 {
				postrouting = append(postrouting, "ip daddr", rule.DstAddr)
				out = append(out, "ip daddr", rule.DstAddr)
				in = append(in, "ip saddr", rule.DstAddr)
			} else {
				postrouting = append(postrouting, "ip6 daddr", rule.DstAddr)
				out = append(out, "ip6 daddr", rule.DstAddr)
				in = append(in, "ip6 saddr", rule.DstAddr)
			}
		}

		if rule.Proto == "icmp" {
			postrouting = append(postrouting, rule.Proto)
			out = append(out, rule.Proto)
			in = append(in, rule.Proto)

			if rule.ICMPType != "" {
				postrouting = append(postrouting, "type", rule.ICMPType)
				out = append(out, "type", rule.ICMPType)
				typ, err := oppositeICMPType(rule.ICMPType)
				if err != nil {
					return err
				}
				in = append(in, "type", typ)
			}
		} else {
			if rule.SrcPort != "" {
				postrouting = append(postrouting, rule.Proto, "sport", string(rule.SrcPort))
				out = append(out, rule.Proto, "sport", string(rule.SrcPort))
				in = append(in, rule.Proto, "dport", string(rule.SrcPort))
			}

			postrouting = append(postrouting, rule.Proto)
			out = append(out, rule.Proto)
			in = append(in, rule.Proto)
			if rule.DstPort != "" {
				postrouting = append(postrouting, "dport", string(rule.DstPort))
				out = append(out, rule.Proto, "dport", string(rule.DstPort))
				in = append(in, rule.Proto, "sport", string(rule.DstPort))
			}
		}

		postrouting = append(postrouting, "ct state new")
		postrouting = append(postrouting, rule.Action)
		postrouting = append(postrouting, rule.Flags...)
		out = append(out, "ct state new,established accept")
		in = append(in, "ct state established accept")

		ctx.PostroutingRules = append(ctx.PostroutingRules, strings.Join(postrouting, " "))
		ctx.ForwardRules = append(ctx.ForwardRules, strings.Join(out, " "))
		ctx.ForwardRules = append(ctx.ForwardRules, strings.Join(in, " "))
	}

	return nil
}

func isIPv4(s string) (bool, error) {
	ip, _, err := net.ParseCIDR(s)
	if err != nil {
		ip = net.ParseIP(s)
	}

	if ip == nil {
		return false, errors.Errorf("%s is not a valid IP", s)
	}

	return ip.To4() != nil, nil
}

func oppositeICMPType(t string) (string, error) {
	switch t {
	case "echo-request":
		return "echo-reply", nil
	case "echo-reply":
		return "echo-request", nil
	}
	return "", errors.Errorf("unsupported icmp type: %s", t)
}
