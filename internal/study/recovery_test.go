package study

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestSettledServiceRecovery(t *testing.T) {
	if _, e := exec.LookPath("jq"); e != nil {
		t.Skip("jq required by service")
	}
	for _, fixture := range []struct {
		status, phase string
		ready         bool
	}{
		{"passed", "", true}, {"running", "", false}, {"error", "", false},
		{"running", "settled-v1", true}, {"error", "settled-v1", true},
	} {
		var input strings.Builder
		enc := json.NewEncoder(&input)
		for i := 0; i < 15; i++ {
			enc.Encode(map[string]string{"status": "passed"})
		}
		enc.Encode(map[string]string{"status": fixture.status, "evaluation_phase": fixture.phase})
		cmd := exec.Command("jq", "-e", "-s", "-f", "../../configs/settled-ready.jq")
		cmd.Stdin = strings.NewReader(input.String())
		if e := cmd.Run(); (e == nil) != fixture.ready {
			t.Fatalf("%+v: %v", fixture, e)
		}
	}
}
