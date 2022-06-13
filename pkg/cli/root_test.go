package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/google/go-cmp/cmp"
)

func TestInitCfg(t *testing.T) {
	cfgFile = filepath.Join(t.TempDir(), ".jvscli.yaml")

	if err := os.WriteFile(cfgFile, []byte(`server: https://example.com
`), fs.ModePerm); err != nil {
		t.Fatalf("failed to prepare test config file: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(cfgFile); err != nil {
			t.Logf("failed to cleanup test config file: %v", err)
		}
	})

	initCfg()

	wantCfg := &config.CLIConfig{
		Version: 1,
		Server:  "https://example.com",
	}
	if diff := cmp.Diff(wantCfg, cfg); diff != "" {
		t.Errorf("CLI config loaded (-want,+got):\n%s", diff)
	}
}
