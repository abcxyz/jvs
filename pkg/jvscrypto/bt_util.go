package jvscrypto

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"cloud.google.com/go/bigtable"
)

// These states are independent of KMS states. This allows us to distinguish which key (regardless of KMS state)
// to use when signing.
type VersionState int64

const (
	VER_STATE_PRIMARY VersionState = iota
	VER_STATE_NEW
	VER_STATE_OLD
	VER_STATE_UNKOWN
)

func (v VersionState) String() string {
	switch v {
	case VER_STATE_PRIMARY:
		return "PRIMARY"
	case VER_STATE_NEW:
		return "NEW"
	}
	return "UNKNOWN"
}

func GetVersionState(s string) VersionState {
	switch s {
	case "PRIMARY":
		return VER_STATE_PRIMARY
	case "NEW":
		return VER_STATE_NEW
	}
	return VER_STATE_UNKOWN
}

func GetActiveVersionStates(ctx context.Context, client *bigtable.Client) (map[string]VersionState, error) {
	tbl := client.Open("jvs-certificates.certificate-states")
	var vers map[string]VersionState
	err := tbl.ReadRows(ctx, bigtable.RowList{"name", "state"}, func(row bigtable.Row) bool {
		vers[row.Key()] = GetVersionState(string(row["version-info"][0].Value))
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("issue while reading from bigtable: %w", err)
	}
	return vers, nil
}

func WriteVersionState(ctx context.Context, client *bigtable.Client, versionName string, state VersionState) error {
	tbl := client.Open("jvs-certificates.certificate-states")
	timestamp := bigtable.Now()

	mut := bigtable.NewMutation()
	mut.Set("version-info", "state", timestamp, []byte(state.String()))

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, int64(1))

	if err := tbl.Apply(ctx, versionName, mut); err != nil {
		return fmt.Errorf("couldn't apply change to bigtable: %v", err)
	}
	return nil
}

func RemoveVersionState(ctx context.Context, client *bigtable.Client, versionName string) error {
	tbl := client.Open("jvs-certificates.certificate-states")

	mut := bigtable.NewMutation()
	mut.DeleteRow()

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, int64(1))

	if err := tbl.Apply(ctx, versionName, mut); err != nil {
		return fmt.Errorf("couldn't apply change to bigtable: %v", err)
	}
	return nil
}
