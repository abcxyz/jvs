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
