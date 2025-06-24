package qubes

import (
	"github.com/illikainen/orch/src/fact"
)

var facts *fact.OS

func IsDom0() (bool, error) {
	if facts == nil {
		tmp, err := fact.GatherOSFacts()
		if err != nil {
			return false, err
		}
		facts = tmp
	}

	return facts.Name == "qubes", nil
}
