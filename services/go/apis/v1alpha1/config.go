package v1alpha1

const (
	// Version of the API and config.
	Version = "v1alpha1"

	// Audit rule directive options.
	AuditRuleDirectiveDefault            = "AUDIT"
	AuditRuleDirectiveRequestOnly        = "AUDIT_REQUEST_ONLY"
	AuditRuleDirectiveRequestAndResponse = "AUDIT_REQUEST_AND_RESPONSE"
)

// Config is the full audit client config.
type Config struct {
	// Version is the version of the config.
	Version string `yaml:"version,omitempty" env:"VERSION,overwrite"`

	// Crypto variables
	KeyTTLDays          uint64 `yaml:"key_ttl_days,omitempty"`
	PropagationTimeMinutes uint64 `yaml:"propagation_time_minutes,omitempty"`
	GracePeriodMinutes uint64 `yaml:"grace_period_minutes,omitempty"`
	DisabledPeriodDays uint64 `yaml:"disabled_period_days,omitempty"`
}

// Validate checks if the config is valid.
func (cfg *Config) Validate() error {
	return nil
}

// SetDefault sets default for the config.
func (cfg *Config) SetDefault() {
	// TODO: set defaults for other fields if necessary.
	if cfg.Version == "" {
		cfg.Version = Version
	}
}

func (cfg *Config) GetRotationAgeSeconds() uint64 {
	ttlSeconds := cfg.KeyTTLDays * 24 * 60 * 60
	graceSeconds := cfg.GracePeriodMinutes * 60
	return ttlSeconds - graceSeconds
}

func (cfg *Config) GetDisableAgeSeconds() uint64 {
	return cfg.KeyTTLDays * 24 * 60 * 60
}

func (cfg *Config) GetDestroyAgeSeconds() uint64 {
	ttlSeconds := cfg.KeyTTLDays * 24 * 60 * 60
	disabledPeriod := cfg.DisabledPeriodDays * 24 * 60 * 60
	return ttlSeconds + disabledPeriod
}