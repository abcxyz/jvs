package jvscrypto

// VersionState is independent of KMS states. This allows us to distinguish which key (regardless of KMS state)
// to use when signing. These are stored in the key labels.
type VersionState int64

const (
	VersionStatePrimary VersionState = iota
	VersionStateNew
	VersionStateOld
	VersionStateUnknown
)

func (v VersionState) String() string {
	switch v {
	case VersionStatePrimary:
		return "primary"
	case VersionStateNew:
		return "new"
	case VersionStateOld:
		return "old"
	}
	return "unknown"
}

// GetVersionState converts a string to a VersionState.
func GetVersionState(s string) VersionState {
	switch s {
	case "primary":
		return VersionStatePrimary
	case "new":
		return VersionStateNew
	case "old":
		return VersionStateOld
	}
	return VersionStateUnknown
}
