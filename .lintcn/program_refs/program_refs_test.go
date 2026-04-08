package program_refs

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/ast"
)

func TestNormalizeSymbolSkipsAliasWithoutDeclaration(t *testing.T) {
	t.Parallel()

	alias := &ast.Symbol{Flags: ast.SymbolFlagsAlias, Name: "broken-alias"}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("normalizeSymbol panicked for alias without declaration: %v", recovered)
		}
	}()

	if got := normalizeSymbol(nil, alias); got != alias {
		t.Fatalf("expected alias symbol to be returned unchanged, got %#v", got)
	}
}
