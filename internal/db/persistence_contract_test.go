package db

import (
	"context"
	"strings"
	"testing"
)

func TestRequiredPersistenceObjectsAreUniqueAndQualified(t *testing.T) {
	seen := make(map[string]struct{}, len(RequiredPersistenceObjects))
	for _, objectName := range RequiredPersistenceObjects {
		if objectName == "" || !strings.Contains(objectName, ".") {
			t.Fatalf("persistence object is not schema-qualified: %q", objectName)
		}
		if _, exists := seen[objectName]; exists {
			t.Fatalf("duplicate persistence object: %q", objectName)
		}
		seen[objectName] = struct{}{}
	}
}

func TestPersistenceContractFeaturesAreDeclared(t *testing.T) {
	if len(RequiredPersistenceFunctions) == 0 || len(RequiredPersistenceColumns) == 0 {
		t.Fatal("persistence adapter features must be part of the contract")
	}
	for _, functionName := range RequiredPersistenceFunctions {
		if functionName == "" || !strings.Contains(functionName, "(") {
			t.Fatalf("invalid persistence function: %q", functionName)
		}
	}
	for _, columnName := range RequiredPersistenceColumns {
		if strings.Count(columnName, ".") != 2 {
			t.Fatalf("column is not schema-qualified: %q", columnName)
		}
	}
}

func TestMissingPersistenceObjectsRequiresPool(t *testing.T) {
	if _, err := (*Pool)(nil).MissingPersistenceObjects(context.Background()); err == nil {
		t.Fatal("expected an unconfigured pool error")
	}
	if err := (*Pool)(nil).CheckPersistenceContract(context.Background()); err == nil {
		t.Fatal("expected an unconfigured pool error")
	}
}
