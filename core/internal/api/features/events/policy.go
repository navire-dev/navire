package events

type Limits struct {
	MaxDataKeys    int
	MaxKeyLen      int
	MaxStrValLen   int
	MaxSlugLen     int
	MaxTenantIDLen int
}

var GlobalLimits = DefaultLimits()

func DefaultLimits() Limits {
	return Limits{MaxDataKeys: 64, MaxKeyLen: 64, MaxStrValLen: 4096, MaxSlugLen: 80, MaxTenantIDLen: 80}
}
