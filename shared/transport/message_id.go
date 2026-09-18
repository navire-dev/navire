package transport

// MessageID is an opaque identifier assigned by the active message transport.
// Callers must not depend on its format or ordering semantics.
type MessageID string
