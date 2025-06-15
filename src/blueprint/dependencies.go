package blueprint

import (
	"github.com/illikainen/go-utils/src/seq"
)

type Dependencies map[string][]string

func (d *Dependencies) Filter(hosts []string) Dependencies {
	deps := Dependencies{}
	for host, depHosts := range *d {
		deps[host] = seq.Filter(depHosts, hosts...)
	}

	return deps
}

func (d *Dependencies) FindCircularDependencies() (bool, string) {
	visited := map[string]bool{}
	cur := map[string]bool{}

	for host := range *d {
		if circular := d.visitCircularDependencies(host, visited, cur); circular {
			return circular, host
		}
	}

	return false, ""
}

func (d *Dependencies) visitCircularDependencies(host string, visited map[string]bool,
	cur map[string]bool) bool {
	if cur[host] {
		return true
	}

	if visited[host] {
		return false
	}

	cur[host] = true
	for _, depHost := range (*d)[host] {
		if circular := d.visitCircularDependencies(depHost, visited, cur); circular {
			return true
		}
	}

	cur[host] = false
	visited[host] = true
	return false
}
