// Package program_refs builds reusable cross-file symbol and call reference
// indexes over the current TypeScript program. Rules can reuse this instead of
// hand-rolling whole-program traversals every time they need callers or symbol
// usages.
package program_refs

import (
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type Reference struct {
	Node       *ast.Node
	SourceFile *ast.SourceFile
}

type FindOptions struct {
	IncludeDeclarations bool
	ExcludeWithin       *ast.Node
}

type index struct {
	symbolRefs map[ast.SymbolId][]Reference
	callRefs   map[ast.SymbolId][]Reference
}

type cacheEntry struct {
	once  sync.Once
	index *index
}

var indexCache sync.Map

func FindSymbolReferences(program *compiler.Program, typeChecker *checker.Checker, target *ast.Symbol, options FindOptions) []Reference {
	idx := getIndex(program, typeChecker)
	if idx == nil {
		return nil
	}
	return filterReferences(idx.symbolRefs[symbolID(typeChecker, target)], options)
}

func FindCallReferences(program *compiler.Program, typeChecker *checker.Checker, target *ast.Symbol, options FindOptions) []Reference {
	idx := getIndex(program, typeChecker)
	if idx == nil {
		return nil
	}
	return filterReferences(idx.callRefs[symbolID(typeChecker, target)], options)
}

func SymbolAtLocation(typeChecker *checker.Checker, node *ast.Node) *ast.Symbol {
	if typeChecker == nil || node == nil {
		return nil
	}
	return normalizeSymbol(typeChecker, typeChecker.GetSymbolAtLocation(node))
}

func getIndex(program *compiler.Program, typeChecker *checker.Checker) *index {
	if program == nil || typeChecker == nil {
		return nil
	}

	entryAny, _ := indexCache.LoadOrStore(typeChecker, &cacheEntry{})
	entry := entryAny.(*cacheEntry)
	entry.once.Do(func() {
		entry.index = buildIndex(program, typeChecker)
	})
	return entry.index
}

func buildIndex(program *compiler.Program, typeChecker *checker.Checker) *index {
	idx := &index{
		symbolRefs: map[ast.SymbolId][]Reference{},
		callRefs:   map[ast.SymbolId][]Reference{},
	}

	for _, sourceFile := range program.SourceFiles() {
		if !shouldTraverseFile(program, sourceFile) {
			continue
		}
		walkNode(&sourceFile.Node, func(node *ast.Node) {
			idx.recordSymbolReference(typeChecker, sourceFile, node)
			idx.recordCallReference(typeChecker, sourceFile, node)
		})
	}

	return idx
}

func shouldTraverseFile(program *compiler.Program, sourceFile *ast.SourceFile) bool {
	if sourceFile == nil || sourceFile.IsDeclarationFile {
		return false
	}
	if utils.IsSourceFileDefaultLibrary(program, sourceFile) || program.IsSourceFileFromExternalLibrary(sourceFile) {
		return false
	}
	return !strings.Contains(sourceFile.FileName(), "/node_modules/")
}

func walkNode(node *ast.Node, visit func(*ast.Node)) {
	if node == nil {
		return
	}
	visit(node)
	node.ForEachChild(func(child *ast.Node) bool {
		walkNode(child, visit)
		return false
	})
}

func (idx *index) recordSymbolReference(typeChecker *checker.Checker, sourceFile *ast.SourceFile, node *ast.Node) {
	if node == nil || !ast.IsIdentifier(node) {
		return
	}
	symbol := SymbolAtLocation(typeChecker, node)
	if symbol == nil {
		return
	}
	id := ast.GetSymbolId(symbol)
	idx.symbolRefs[id] = append(idx.symbolRefs[id], Reference{Node: node, SourceFile: sourceFile})
}

func (idx *index) recordCallReference(typeChecker *checker.Checker, sourceFile *ast.SourceFile, node *ast.Node) {
	if node == nil || !ast.IsCallExpression(node) {
		return
	}
	call := node.AsCallExpression()
	if call == nil || call.Expression == nil {
		return
	}
	symbol := calleeSymbol(typeChecker, call.Expression)
	if symbol == nil {
		return
	}
	id := ast.GetSymbolId(symbol)
	idx.callRefs[id] = append(idx.callRefs[id], Reference{Node: node, SourceFile: sourceFile})
}

func calleeSymbol(typeChecker *checker.Checker, expression *ast.Node) *ast.Symbol {
	expression = ast.SkipParentheses(expression)
	if expression == nil {
		return nil
	}
	if ast.IsPropertyAccessExpression(expression) {
		return SymbolAtLocation(typeChecker, expression.AsPropertyAccessExpression().Name())
	}
	return SymbolAtLocation(typeChecker, expression)
}

func filterReferences(references []Reference, options FindOptions) []Reference {
	if len(references) == 0 {
		return nil
	}

	filtered := make([]Reference, 0, len(references))
	for _, reference := range references {
		if reference.Node == nil {
			continue
		}
		if !options.IncludeDeclarations && ast.IsDeclarationName(reference.Node) {
			continue
		}
		if options.ExcludeWithin != nil && isWithinNode(reference.Node, options.ExcludeWithin) {
			continue
		}
		filtered = append(filtered, reference)
	}

	return filtered
}

func isWithinNode(node *ast.Node, ancestor *ast.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current == ancestor {
			return true
		}
	}
	return false
}

func symbolID(typeChecker *checker.Checker, symbol *ast.Symbol) ast.SymbolId {
	symbol = normalizeSymbol(typeChecker, symbol)
	if symbol == nil {
		return 0
	}
	return ast.GetSymbolId(symbol)
}

func normalizeSymbol(typeChecker *checker.Checker, symbol *ast.Symbol) *ast.Symbol {
	for symbol != nil && utils.IsSymbolFlagSet(symbol, ast.SymbolFlagsAlias) {
		// Some real-world programs surface alias-flagged symbols without an alias
		// declaration. typescript-go panics if getImmediateAliasedSymbol is called on
		// those, so stop normalizing and keep the original symbol instead.
		if checker.Checker_getDeclarationOfAliasSymbol(typeChecker, symbol) == nil {
			break
		}
		next := checker.Checker_getImmediateAliasedSymbol(typeChecker, symbol)
		if next == nil || next == symbol {
			break
		}
		symbol = next
	}
	return symbol
}
