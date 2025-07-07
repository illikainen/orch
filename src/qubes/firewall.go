package qubes

import (
	"fmt"
	"net"
	"strings"

	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/kballard/go-shellquote"
	"golang.org/x/exp/slices"

	"github.com/illikainen/orch/src/tasks/outputs"
)

type Firewall struct {
	Rules []FirewallRule `json:"rule" hcl:"rule"`
}

func (f *Firewall) Apply(name string, dryRun bool) (int, []string, error) {
	exists, err := Check(name)
	if err != nil {
		return 0, nil, err
	}
	if !exists {
		return outputs.StatusIndeterminate, nil, nil
	}

	p, err := process.Exec(&process.ExecOptions{
		Command: []string{"qvm-firewall", "--raw", "--", name},
	})
	if err != nil {
		return 0, nil, err
	}
	oldRules := stringx.SplitLines(string(p.Stdout))

	newRules := []string{}
	for _, rule := range f.Rules {
		newRules = append(newRules, rule.String())
	}

	if slices.Compare(oldRules, newRules) == 0 {
		return outputs.StatusUnchanged, nil, nil
	}

	changes := []string{}
	for _, rule := range oldRules {
		changes = append(changes, "- "+rule)
	}

	for _, rule := range newRules {
		elts, err := shellquote.Split(rule)
		if err != nil {
			return 0, nil, err
		}

		if !dryRun {
			_, err = process.Exec(&process.ExecOptions{
				Command: append([]string{"qvm-firewall", "--", name, "add"}, elts...),
			})
			if err != nil {
				return 0, nil, err
			}
		}
		changes = append(changes, "+ "+rule)
	}

	if !dryRun {
		for i := 0; i < len(oldRules); i++ {
			_, err = process.Exec(&process.ExecOptions{
				Command: []string{"qvm-firewall", name, "del", "--rule-no=0"},
			})
			if err != nil {
				return 0, nil, err
			}
		}
	}

	return 0, changes, nil
}

type FirewallRule struct {
	Action        string `json:"action"         hcl:"action"`
	Protocol      string `json:"proto"          hcl:"proto,optional"`
	DstHost       string `json:"dst_host"       hcl:"dst_host,optional"`
	DstPorts      string `json:"dst_ports"      hcl:"dst_ports,optional"`
	SpecialTarget string `json:"special_target" hcl:"special_target,optional"`
	Comment       string `json:"comment"        hcl:"comment,optional"`
}

func (f *FirewallRule) String() string {
	s := fmt.Sprintf("action=%s", f.Action)

	if f.Protocol != "" {
		s += fmt.Sprintf(" proto=%s", f.Protocol)
	}

	if f.DstHost != "" {
		dsthost := f.DstHost
		if !strings.Contains(dsthost, "/") {
			dsthost += "/32"
		}

		ip, _, err := net.ParseCIDR(dsthost)
		if err == nil {
			if ip.To4() == nil {
				s += fmt.Sprintf(" dst4=%s", dsthost)
			} else {
				s += fmt.Sprintf(" dst6=%s", dsthost)
			}
		}
	}

	if f.DstPorts != "" {
		dstports := f.DstPorts
		if !strings.Contains(dstports, "-") {
			// This is because the output from `qvm-firewall --raw`
			// that we compare against normalizes single ports
			// (e.g., 22) to port ranges (e.g., 22-22)
			dstports += fmt.Sprintf("-%s", dstports)
		}

		s += fmt.Sprintf(" dstports=%s", dstports)
	}

	if f.SpecialTarget != "" {
		s += fmt.Sprintf(" specialtarget=%s", f.SpecialTarget)
	}

	if f.Comment != "" {
		s += fmt.Sprintf(" comment=%s", f.Comment)
	}

	return s
}
