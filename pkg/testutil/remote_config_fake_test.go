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
	keyCfg, err := NewFakeRemoteConfig(testYAMLStr(t, jvsKeyConfig), "yaml")
	if err != nil {
		t.Fatalf("failed to create fake remote config: %v", err)
	}

	var gotJVSKeyConfig config.JVSKeyConfig
	if err = keyCfg.Unmarshal(ctx, &gotJVSKeyConfig); err != nil {
		t.Errorf("unexpected error when unmarshaling : %v", err)
		return
	}
	if diff := cmp.Diff(gotJVSKeyConfig, jvsKeyConfig); diff != "" {
		t.Errorf("Got diff (-want, +got): %v", diff)
	}
}

func TestGetAndSet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	keyName := "projects/fake_project/locations/us-west-1/keyRings/test-key-ring/cryptoKeys/test-key"
	jvsKeyConfig := config.JVSKeyConfig{KeyName: keyName}
	keyCfg, err := NewFakeRemoteConfig(testYAMLStr(t, jvsKeyConfig), "yaml")
	if err != nil {
		t.Fatalf("failed to create fake remote config: %v", err)
	}
	gotKeyName, err := keyCfg.Get(ctx, config.JVSKeyNameField)
	if err != nil {
		t.Errorf("unexpected error when getting field : %v", err)
		return
	}
	if diff := cmp.Diff(gotKeyName, keyName); diff != "" {
		t.Errorf("Got diff (-want, +got): %v", diff)
	}

	updatedFakeKeyName := "test-key"
	if err = keyCfg.Set(ctx, config.JVSKeyNameField, updatedFakeKeyName); err != nil {
		t.Errorf("unexpected error when setting field : %v", err)
		return
	}
	gotUpdatedKeyName, err := keyCfg.Get(ctx, config.JVSKeyNameField)
	if err != nil {
		t.Errorf("unexpected error when getting field : %v", err)
		return
	}
	if diff := cmp.Diff(gotUpdatedKeyName, updatedFakeKeyName); diff != "" {
		t.Errorf("Got diff (-want, +got): %v", diff)
	}
}

func testYAMLStr(tb testing.TB, in interface{}) string {
	tb.Helper()
	inBytes, err := yaml.Marshal(in)
	if err != nil {
		tb.Fatalf("failed to marshal: %v", err)
	}
	return string(inBytes)
}
