package query

import (
	"context"
	"testing"
)

func TestParseCQL(t *testing.T) {
	query, err := ParseCQL(`SELECT modules WHERE fan_in > 0 AND name CONTAINS "app/"`)
	if err != nil {
		t.Fatalf("parse cql: %v", err)
	}
	if query.Target != "modules" {
		t.Fatalf("expected target modules, got %q", query.Target)
	}
	if len(query.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(query.Conditions))
	}
}

func TestParseCQL_Invalid(t *testing.T) {
	if _, err := ParseCQL("DELETE FROM modules"); err == nil {
		t.Fatal("expected invalid CQL query to fail")
	}
}

func TestService_ExecuteCQL(t *testing.T) {
	svc := NewService(seedGraph(), nil, "default")

	rows, err := svc.ExecuteCQL(context.Background(), `SELECT modules WHERE dependency_count >= 1 AND fan_out >= 1`, 0)
	if err != nil {
		t.Fatalf("execute cql: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two modules with outgoing deps, got %d", len(rows))
	}
	if rows[0].Name != "app/a" || rows[1].Name != "app/b" {
		t.Fatalf("unexpected module set: %+v", rows)
	}
}
