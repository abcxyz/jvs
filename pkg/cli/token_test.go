package cli

import (
	"testing"

	"github.com/abcxyz/pkg/testutil"
)

func TestRunTokenCmd(t *testing.T) {
	err := runTokenCmd(nil, nil)
	wantErrStr := "not implemented"
	if diff := testutil.DiffErrString(err, wantErrStr); diff != "" {
		t.Errorf("unexpected error: %s", diff)
	}
}
