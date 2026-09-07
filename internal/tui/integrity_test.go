package tui

import (
	"context"
	"strings"
	"testing"

	db "github.com/sahas/readit/internal/db/sqlc"
)

func TestSecurity_EmptyPubkeyRejected(t *testing.T) {
	// Empty fingerprint should be rejected immediately by lookupUserCmd
	m := NewModel(context.Background(), nil, "", nil)
	cmd := m.lookupUserCmd()
	msg := cmd()
	em1, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("expected errMsg for empty fingerprint in lookupUserCmd, got %T", msg)
	}
	if !strings.Contains(em1.err.Error(), "public key authentication required") {
		t.Errorf("unexpected error message: %v", em1.err)
	}

	// Whitespace fingerprint should also be rejected
	m2 := NewModel(context.Background(), nil, "   ", nil)
	cmd2 := m2.lookupUserCmd()
	msg2 := cmd2()
	em2, ok := msg2.(errMsg)
	if !ok {
		t.Fatalf("expected errMsg for whitespace fingerprint, got %T", msg2)
	}
	if !strings.Contains(em2.err.Error(), "public key authentication required") {
		t.Errorf("unexpected error message: %v", em2.err)
	}

	// createUserCmd with empty fingerprint should also be rejected
	cmd3 := m.createUserCmd("alice")
	msg3 := cmd3()
	em3, ok := msg3.(errMsg)
	if !ok {
		t.Fatalf("expected errMsg for empty fingerprint in createUserCmd, got %T", msg3)
	}
	if !strings.Contains(em3.err.Error(), "public key authentication required") {
		t.Errorf("unexpected error message: %v", em3.err)
	}
}

func TestTransactional_ExecTxNilPoolFallback(t *testing.T) {
	m := NewModel(context.Background(), nil, "fp1234567890", nil)
	executed := false
	err := m.execTx(context.Background(), func(q *db.Queries) error {
		executed = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error in execTx with nil pool: %v", err)
	}
	if !executed {
		t.Errorf("expected fn to be executed in execTx fallback")
	}
}
