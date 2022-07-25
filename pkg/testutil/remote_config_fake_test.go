package testutil

import (
	"context"
	"testing"

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	keyName := "projects/fake_project/locations/us-west-1/keyRings/test-key-ring/cryptoKeys/test-key"
	jvsKeyConfig := config.JVSKeyConfig{KeyName: keyName}
	keyConfig, err := NewFakeRemoteConfig(testYAMLStr(ctx, t, jvsKeyConfig), "unit_test_fake_remote_config_unmarshal.yaml", "yaml")
	if err != nil {
		t.Fatalf("failed to create fake remote config: %v", err)
	}

	var gotJVSKeyConfig config.JVSKeyConfig
	if err = keyConfig.Unmarshal(ctx, &gotJVSKeyConfig); err != nil {
		t.Errorf("unexpected error when unmarshaling : %v", err)
		return
	}
	if diff := cmp.Diff(gotJVSKeyConfig, jvsKeyConfig); diff != "" {
		t.Errorf("Got diff (-want, +got): %v", diff)
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	keyName := "projects/fake_project/locations/us-west-1/keyRings/test-key-ring/cryptoKeys/test-key"
	jvsKeyConfig := config.JVSKeyConfig{KeyName: keyName}
	keyConfig, err := NewFakeRemoteConfig(testYAMLStr(ctx, t, jvsKeyConfig), "unit_test_fake_remote_config_get.yaml", "yaml")
	if err != nil {
		t.Fatalf("failed to create fake remote config: %v", err)
	}
	gotKeyName, err := keyConfig.Get(ctx, "key_name")
	if err != nil {
		t.Errorf("unexpected error when getting field : %v", err)
		return
	}
	if diff := cmp.Diff(gotKeyName, keyName); diff != "" {
		t.Errorf("Got diff (-want, +got): %v", diff)
	}
}

func TestSet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	keyName := "projects/fake_project/locations/us-west-1/keyRings/test-key-ring/cryptoKeys/test-key"
	jvsKeyConfig := config.JVSKeyConfig{KeyName: keyName}
	keyConfig, err := NewFakeRemoteConfig(testYAMLStr(ctx, t, jvsKeyConfig), "unit_test_fake_remote_config_set.yaml", "yaml")
	if err != nil {
		t.Fatalf("failed to create fake remote config: %v", err)
	}

	if err = keyConfig.Set(ctx, "key_name", "test-key"); err != nil {
		t.Errorf("unexpected error when setting field : %v", err)
		return
	}
	gotKeyName, err := keyConfig.Get(ctx, "key_name")
	if err != nil {
		t.Errorf("unexpected error when getting field : %v", err)
		return
	}
	if diff := cmp.Diff(gotKeyName, "test-key"); diff != "" {
		t.Errorf("Got diff (-want, +got): %v", diff)
	}
}

func testYAMLStr(ctx context.Context, tb testing.TB, in interface{}) string {
	tb.Helper()
	inBytes, err := yaml.Marshal(in)
	if err != nil {
		tb.Fatalf("failed to marshal: %v", err)
	}
	return string(inBytes)
}
