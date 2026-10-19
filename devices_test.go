package devices_test

import (
	"testing"

	devices "github.com/avanha/pmaas-bluetooth-devices"
)

func TestAllModelsAreNamedAndTheirRulesAreValid(t *testing.T) {
	models := devices.All()
	if len(models) < 2 {
		t.Fatalf("only %d models", len(models))
	}

	seen := map[string]bool{}

	for _, m := range models {
		if m.Name() == "" || seen[m.Name()] || m.Kind() == 0 || m.New() == nil {
			t.Errorf("%q is not a good model", m.Name())
		}

		seen[m.Name()] = true

		for _, r := range m.Rules() {
			if err := r.Validate(); err != nil {
				t.Errorf("%s: %v", m.Name(), err)
			}
		}
	}
}
