package jvscrypto

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"cloud.google.com/go/bigtable"
)

// VersionState is independent of KMS states. This allows us to distinguish which key (regardless of KMS state)
// to use when signing.
type VersionState int64

const (
	VersionStatePrimary VersionState = iota
	VersionStateNew
	VersionStateOld
	VersionStateUnknown
)

const (
	TableName  = "certificate-states"
	FamilyName = "version-info"
)

func (v VersionState) String() string {
	switch v {
	case VersionStatePrimary:
		return "PRIMARY"
	case VersionStateNew:
		return "NEW"
	case VersionStateOld:
		return "OLD"
	}
	return "UNKNOWN"
}

// GetVersionState converts a string to a VersionState.
func GetVersionState(s string) VersionState {
	switch s {
	case "PRIMARY":
		return VersionStatePrimary
	case "NEW":
		return VersionStateNew
	case "OLD":
		return VersionStateOld
	}
	return VersionStateUnknown
}

// GetActiveVersionStates returns a map from key version name to VersionState from BigTable.
func GetActiveVersionStates(ctx context.Context, client *bigtable.Client) (map[string]VersionState, error) {
	tbl := client.Open(TableName)
	vers := make(map[string]VersionState)
	err := tbl.ReadRows(ctx, bigtable.RowRange{}, func(row bigtable.Row) bool {
		vers[row.Key()] = GetVersionState(string(row[FamilyName][0].Value))
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("issue while reading from bigtable: %w", err)
	}
	return vers, nil
}

// WriteVersionState writes a key version name and VersionState to BigTable.
func WriteVersionState(ctx context.Context, client *bigtable.Client, versionName string, state VersionState) error {
	tbl := client.Open(TableName)
	timestamp := bigtable.Now()

	mut := bigtable.NewMutation()
	mut.Set(FamilyName, "state", timestamp, []byte(state.String()))

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, int64(1))

	if err := tbl.Apply(ctx, versionName, mut); err != nil {
		return fmt.Errorf("couldn't apply change to bigtable: %v", err)
	}
	return nil
}

// RemoveVersion removes the version from BigTable.
func RemoveVersion(ctx context.Context, client *bigtable.Client, versionName string) error {
	tbl := client.Open(TableName)

	mut := bigtable.NewMutation()
	mut.DeleteRow()

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, int64(1))

	if err := tbl.Apply(ctx, versionName, mut); err != nil {
		return fmt.Errorf("couldn't apply change to bigtable: %v", err)
	}
	return nil
}
