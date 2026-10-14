package db

import "testing"

func TestCreateIdentity(t *testing.T) {
	q := New(testTx(t))

	traits := Traits{
		Email:     "test@test.com",
		FirstName: "Test1",
		LastName:  "Test",
	}
	got, err := q.CreateIdentity(t.Context(), traits)
	if err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	if got.ID.Version() != 7 {
		t.Errorf("ID version = %d, want 7", got.ID.Version())
	}
	if got.Traits != traits {
		t.Errorf("Traits = %+v, want %+v", got.Traits, traits)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}
