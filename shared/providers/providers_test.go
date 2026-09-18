package providers

import "testing"

func TestValidateUsesCatalogAndActivation(t *testing.T) {
	if err := Validate(string(Gotify)); err != nil {
		t.Fatalf("Validate(gotify) error = %v", err)
	}
	if err := Validate(string(Slack)); err != nil {
		t.Fatalf("Validate(slack) rejected a known inactive provider: %v", err)
	}
	if err := Validate(string(Ntfy)); err != nil {
		t.Fatalf("Validate(ntfy) error = %v", err)
	}
	if !IsActive(string(Ntfy)) {
		t.Fatal("IsActive(ntfy) returned false")
	}
	if !IsActive(string(Slack)) {
		t.Fatal("IsActive(slack) returned false")
	}
	if err := ValidateActive(string(Slack)); err != nil {
		t.Fatalf("ValidateActive(slack) rejected an active provider: %v", err)
	}
	if err := Validate("unknown"); err == nil {
		t.Fatal("Validate(unknown) accepted an unknown provider")
	}
}

func TestActiveIDsReturnsActiveCatalogEntries(t *testing.T) {
	active := ActiveIDs()
	want := []ID{Gotify, Discord, Ntfy, Slack}
	if len(active) != len(want) {
		t.Fatalf("ActiveIDs() returned %d entries, want %d", len(active), len(want))
	}
	for index := range want {
		if active[index] != want[index] {
			t.Fatalf("ActiveIDs()[%d] = %q, want %q", index, active[index], want[index])
		}
	}
}
