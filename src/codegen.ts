// Generate Go workspace files for building a custom tsgolint binary.
// Creates:
//   .lintcn/go.work       — workspace for gopls (editor support)
//   .lintcn/go.mod        — module declaration
//   build/go.work          — build workspace in cache dir
//   build/wrapper/go.mod   — wrapper module
//   build/wrapper/main.go  — entry point with all rules

import fs from 'node:fs'
import path from 'node:path'
import type { RuleMetadata } from './discover.ts'

// All replace directives needed from tsgolint's go.mod.
// These redirect shim module paths to local directories inside the tsgolint source.
const SHIM_MODULES = [
  'ast',
  'bundled',
  'checker',
  'compiler',
  'core',
  'lsp/lsproto',
  'parser',
  'project',
  'scanner',
  'tsoptions',
  'tspath',
  'vfs',
  'vfs/cachedvfs',
  'vfs/osvfs',
] as const

// Built-in tsgolint rules — all 50+ from cmd/tsgolint/main.go.
// Each entry: [package_path_suffix, exported_var_name]
const BUILTIN_RULES: [string, string][] = [
  ['await_thenable', 'AwaitThenableRule'],
  ['consistent_return', 'ConsistentReturnRule'],
  ['consistent_type_exports', 'ConsistentTypeExportsRule'],
  ['dot_notation', 'DotNotationRule'],
  ['no_array_delete', 'NoArrayDeleteRule'],
  ['no_base_to_string', 'NoBaseToStringRule'],
  ['no_confusing_void_expression', 'NoConfusingVoidExpressionRule'],
  ['no_deprecated', 'NoDeprecatedRule'],
  ['no_duplicate_type_constituents', 'NoDuplicateTypeConstituentsRule'],
  ['no_floating_promises', 'NoFloatingPromisesRule'],
  ['no_for_in_array', 'NoForInArrayRule'],
  ['no_implied_eval', 'NoImpliedEvalRule'],
  ['no_meaningless_void_operator', 'NoMeaninglessVoidOperatorRule'],
  ['no_misused_promises', 'NoMisusedPromisesRule'],
  ['no_misused_spread', 'NoMisusedSpreadRule'],
  ['no_mixed_enums', 'NoMixedEnumsRule'],
  ['no_redundant_type_constituents', 'NoRedundantTypeConstituentsRule'],
  ['no_unnecessary_boolean_literal_compare', 'NoUnnecessaryBooleanLiteralCompareRule'],
  ['no_unnecessary_condition', 'NoUnnecessaryConditionRule'],
  ['no_unnecessary_qualifier', 'NoUnnecessaryQualifierRule'],
  ['no_unnecessary_template_expression', 'NoUnnecessaryTemplateExpressionRule'],
  ['no_unnecessary_type_conversion', 'NoUnnecessaryTypeConversionRule'],
  ['no_unnecessary_type_arguments', 'NoUnnecessaryTypeArgumentsRule'],
  ['no_unnecessary_type_parameters', 'NoUnnecessaryTypeParametersRule'],
  ['no_unnecessary_type_assertion', 'NoUnnecessaryTypeAssertionRule'],
  ['no_useless_default_assignment', 'NoUselessDefaultAssignmentRule'],
  ['no_unsafe_argument', 'NoUnsafeArgumentRule'],
  ['no_unsafe_assignment', 'NoUnsafeAssignmentRule'],
  ['no_unsafe_call', 'NoUnsafeCallRule'],
  ['no_unsafe_enum_comparison', 'NoUnsafeEnumComparisonRule'],
  ['no_unsafe_member_access', 'NoUnsafeMemberAccessRule'],
  ['no_unsafe_return', 'NoUnsafeReturnRule'],
  ['no_unsafe_type_assertion', 'NoUnsafeTypeAssertionRule'],
  ['no_unsafe_unary_minus', 'NoUnsafeUnaryMinusRule'],
  ['non_nullable_type_assertion_style', 'NonNullableTypeAssertionStyleRule'],
  ['only_throw_error', 'OnlyThrowErrorRule'],
  ['prefer_find', 'PreferFindRule'],
  ['prefer_includes', 'PreferIncludesRule'],
  ['prefer_optional_chain', 'PreferOptionalChainRule'],
  ['prefer_nullish_coalescing', 'PreferNullishCoalescingRule'],
  ['prefer_promise_reject_errors', 'PreferPromiseRejectErrorsRule'],
  ['prefer_readonly_parameter_types', 'PreferReadonlyParameterTypesRule'],
  ['prefer_regexp_exec', 'PreferRegexpExecRule'],
  ['prefer_readonly', 'PreferReadonlyRule'],
  ['prefer_reduce_type_parameter', 'PreferReduceTypeParameterRule'],
  ['prefer_return_this_type', 'PreferReturnThisTypeRule'],
  ['prefer_string_starts_ends_with', 'PreferStringStartsEndsWithRule'],
  ['promise_function_async', 'PromiseFunctionAsyncRule'],
  ['related_getter_setter_pairs', 'RelatedGetterSetterPairsRule'],
  ['require_array_sort_compare', 'RequireArraySortCompareRule'],
  ['require_await', 'RequireAwaitRule'],
  ['restrict_plus_operands', 'RestrictPlusOperandsRule'],
  ['restrict_template_expressions', 'RestrictTemplateExpressionsRule'],
  ['return_await', 'ReturnAwaitRule'],
  ['strict_boolean_expressions', 'StrictBooleanExpressionsRule'],
  ['strict_void_return', 'StrictVoidReturnRule'],
  ['switch_exhaustiveness_check', 'SwitchExhaustivenessCheckRule'],
  ['unbound_method', 'UnboundMethodRule'],
  ['use_unknown_in_catch_callback_variable', 'UseUnknownInCatchCallbackVariableRule'],
]

function generateReplaceDirectives(tsgolintRelPath: string): string {
  return SHIM_MODULES.map((mod) => {
    return `\tgithub.com/microsoft/typescript-go/shim/${mod} => ${tsgolintRelPath}/shim/${mod}`
  }).join('\n')
}

function generateShimRequires(): string {
  return SHIM_MODULES.map((mod) => {
    return `\tgithub.com/microsoft/typescript-go/shim/${mod} v0.0.0`
  }).join('\n')
}

/** Generate .lintcn/go.work and .lintcn/go.mod for editor/gopls support.
 *
 *  Key learnings from testing:
 *  - Module name MUST be a child path of github.com/typescript-eslint/tsgolint
 *    so Go allows importing internal/ packages across the module boundary.
 *  - go.work must `use` both .tsgolint AND .tsgolint/typescript-go since
 *    tsgolint's own go.work (which does this) is ignored by the outer workspace.
 *  - go.mod should be minimal (no requires) — the workspace resolves everything. */
export function generateEditorGoFiles(lintcnDir: string): void {
  const goWork = `go 1.26

use (
\t.
\t./.tsgolint
\t./.tsgolint/typescript-go
)

replace (
${generateReplaceDirectives('./.tsgolint')}
)
`

  const goMod = `module github.com/typescript-eslint/tsgolint/lintcn-rules

go 1.26
`

  const gitignore = `.tsgolint/
go.work
go.work.sum
go.mod
go.sum
`

  fs.writeFileSync(path.join(lintcnDir, 'go.work'), goWork)
  fs.writeFileSync(path.join(lintcnDir, 'go.mod'), goMod)

  const gitignorePath = path.join(lintcnDir, '.gitignore')
  if (!fs.existsSync(gitignorePath)) {
    fs.writeFileSync(gitignorePath, gitignore)
  }
}

/** Generate build workspace in cache dir for compiling the custom binary */
export function generateBuildWorkspace({
  buildDir,
  tsgolintDir,
  lintcnDir,
  rules,
}: {
  buildDir: string
  tsgolintDir: string
  lintcnDir: string
  rules: RuleMetadata[]
}): void {
  fs.mkdirSync(path.join(buildDir, 'wrapper'), { recursive: true })

  // symlink tsgolint source
  const tsgolintLink = path.join(buildDir, 'tsgolint')
  if (fs.existsSync(tsgolintLink)) {
    fs.rmSync(tsgolintLink, { recursive: true })
  }
  fs.symlinkSync(tsgolintDir, tsgolintLink)

  // symlink user rules
  const rulesLink = path.join(buildDir, 'rules')
  if (fs.existsSync(rulesLink)) {
    fs.rmSync(rulesLink, { recursive: true })
  }
  fs.symlinkSync(path.resolve(lintcnDir), rulesLink)

  // go.work — must include typescript-go submodule and use child module paths
  const goWork = `go 1.26

use (
\t./tsgolint
\t./tsgolint/typescript-go
\t./wrapper
\t./rules
)

replace (
${generateReplaceDirectives('./tsgolint')}
)
`
  fs.writeFileSync(path.join(buildDir, 'go.work'), goWork)

  // wrapper/go.mod — must be child path of tsgolint for internal/ access
  const wrapperGoMod = `module github.com/typescript-eslint/tsgolint/lintcn-wrapper

go 1.26

require (
\tgithub.com/typescript-eslint/tsgolint v0.0.0
\tgithub.com/typescript-eslint/tsgolint/lintcn-rules v0.0.0
)
`
  fs.writeFileSync(path.join(buildDir, 'wrapper', 'go.mod'), wrapperGoMod)

  // wrapper/main.go
  const mainGo = generateMainGo(rules)
  fs.writeFileSync(path.join(buildDir, 'wrapper', 'main.go'), mainGo)
}

/** Generate the main.go that imports all built-in + custom rules.
 *  This is essentially tsgolint's cmd/tsgolint/main.go with custom rules appended
 *  to allRules. We import tsgolint's internal packages directly since go.work
 *  allows cross-module internal imports. */
function generateMainGo(customRules: RuleMetadata[]): string {
  const tsgolintPkg = 'github.com/typescript-eslint/tsgolint'

  // built-in rule imports
  const builtinImports = BUILTIN_RULES.map(([pkg]) => {
    return `\t"${tsgolintPkg}/internal/rules/${pkg}"`
  }).join('\n')

  // built-in rule entries
  const builtinEntries = BUILTIN_RULES.map(([pkg, varName]) => {
    return `\t${pkg}.${varName},`
  }).join('\n')

  // custom rule entries (all in package lintcn via single import)
  const customEntries = customRules.map((r) => {
    return `\tlintcn.${r.varName},`
  }).join('\n')

  // only add lintcn import if there are custom rules
  const lintcnImport = customRules.length > 0 ? '\n\tlintcn "github.com/typescript-eslint/tsgolint/lintcn-rules"' : ''

  return `// Code generated by lintcn. DO NOT EDIT.
package main

import (
\t"bufio"
\t"flag"
\t"fmt"
\t"math"
\t"os"
\t"runtime"
\t"runtime/pprof"
\t"runtime/trace"
\t"slices"
\t"strconv"
\t"strings"
\t"sync"
\t"time"
\t"unicode"

\t"${tsgolintPkg}/internal/diagnostic"
\t"${tsgolintPkg}/internal/linter"
\t"${tsgolintPkg}/internal/rule"
\t"${tsgolintPkg}/internal/utils"

${builtinImports}${lintcnImport}

\t"github.com/microsoft/typescript-go/shim/ast"
\t"github.com/microsoft/typescript-go/shim/bundled"
\t"github.com/microsoft/typescript-go/shim/core"
\t"github.com/microsoft/typescript-go/shim/scanner"
\t"github.com/microsoft/typescript-go/shim/tspath"
\t"github.com/microsoft/typescript-go/shim/vfs/cachedvfs"
\t"github.com/microsoft/typescript-go/shim/vfs/osvfs"
)

var allRules = []rule.Rule{
${builtinEntries}
${customEntries}
}

var allRulesByName = make(map[string]rule.Rule, len(allRules))

func init() {
\tfor _, rule := range allRules {
\t\tallRulesByName[rule.Name] = rule
\t}
}

// Below is copied from tsgolint's cmd/tsgolint/main.go.
// We cannot import it directly because main packages are not importable in Go.

func recordTrace(traceOut string) (func(), error) {
\tif traceOut != "" {
\t\tf, err := os.Create(traceOut)
\t\tif err != nil {
\t\t\treturn nil, fmt.Errorf("error creating trace file: %w", err)
\t\t}
\t\ttrace.Start(f)
\t\treturn func() {
\t\t\ttrace.Stop()
\t\t\tf.Close()
\t\t}, nil
\t}
\treturn func() {}, nil
}

func recordCpuprof(cpuprofOut string) (func(), error) {
\tif cpuprofOut != "" {
\t\tf, err := os.Create(cpuprofOut)
\t\tif err != nil {
\t\t\treturn nil, fmt.Errorf("error creating cpuprof file: %w", err)
\t\t}
\t\terr = pprof.StartCPUProfile(f)
\t\tif err != nil {
\t\t\treturn nil, fmt.Errorf("error starting cpu profiling: %w", err)
\t\t}
\t\treturn func() {
\t\t\tpprof.StopCPUProfile()
\t\t\tf.Close()
\t\t}, nil
\t}
\treturn func() {}, nil
}

const spaces = "                                                                                                    "

func printDiagnostic(d rule.RuleDiagnostic, w *bufio.Writer, comparePathOptions tspath.ComparePathsOptions) {
\tdiagnosticStart := d.Range.Pos()
\tdiagnosticEnd := d.Range.End()
\tdiagnosticStartLine, diagnosticStartColumn := scanner.GetECMALineAndUTF16CharacterOfPosition(d.SourceFile, diagnosticStart)
\tdiagnosticEndline, _ := scanner.GetECMALineAndUTF16CharacterOfPosition(d.SourceFile, diagnosticEnd)
\tlineMap := d.SourceFile.ECMALineMap()
\ttext := d.SourceFile.Text()
\tcodeboxStartLine := max(diagnosticStartLine-1, 0)
\tcodeboxEndLine := min(diagnosticEndline+1, len(lineMap)-1)
\tcodeboxStart := scanner.GetECMAPositionOfLineAndUTF16Character(d.SourceFile, codeboxStartLine, 0)
\tvar codeboxEndColumn int
\tif codeboxEndLine == len(lineMap)-1 {
\t\tcodeboxEndColumn = len(text) - int(lineMap[len(lineMap)-1])
\t} else {
\t\tcodeboxEndColumn = int(lineMap[codeboxEndLine+1]-lineMap[codeboxEndLine]) - 1
\t}
\tcodeboxEnd := scanner.GetECMAPositionOfLineAndUTF16Character(d.SourceFile, codeboxEndLine, core.UTF16Offset(codeboxEndColumn))
\tw.Write([]byte{' ', 0x1b, '[', '7', 'm', 0x1b, '[', '1', 'm', 0x1b, '[', '3', '8', ';', '5', ';', '3', '7', 'm', ' '})
\tw.WriteString(d.RuleName)
\tw.WriteString(" \\x1b[0m — ")
\tmessageLineStart := 0
\tfor i, char := range d.Message.Description {
\t\tif char == '\\n' {
\t\t\tw.WriteString(d.Message.Description[messageLineStart : i+1])
\t\t\tmessageLineStart = i + 1
\t\t\tw.WriteString("    \\x1b[2m│\\x1b[0m")
\t\t\tw.WriteString(spaces[:len(d.RuleName)+1])
\t\t}
\t}
\tif messageLineStart <= len(d.Message.Description) {
\t\tw.WriteString(d.Message.Description[messageLineStart:len(d.Message.Description)])
\t}
\tw.WriteString("\\n  \\x1b[2m╭─┴──────────(\\x1b[0m \\x1b[3m\\x1b[38;5;117m")
\tw.WriteString(tspath.ConvertToRelativePath(d.SourceFile.FileName(), comparePathOptions))
\tw.WriteByte(':')
\tw.WriteString(strconv.Itoa(diagnosticStartLine + 1))
\tw.WriteByte(':')
\tw.WriteString(strconv.Itoa(int(diagnosticStartColumn) + 1))
\tw.WriteString("\\x1b[0m \\x1b[2m)─────\\x1b[0m\\n")
\tindentSize := math.MaxInt
\tline := codeboxStartLine
\tlineIndentCalculated := false
\tlastNonSpaceIndex := -1
\tlineStarts := make([]int, 13)
\tlineEnds := make([]int, 13)
\tif codeboxEndLine-codeboxStartLine >= len(lineEnds) {
\t\tw.WriteString("  \\x1b[2m│\\x1b[0m  Error range is too big. Skipping code block printing.\\n  \\x1b[2m╰────────────────────────────────\\x1b[0m\\n\\n")
\t\treturn
\t}
\tfor i, char := range text[codeboxStart:codeboxEnd] {
\t\tif char == '\\n' {
\t\t\tif line != codeboxEndLine {
\t\t\t\tlineIndentCalculated = false
\t\t\t\tlineEnds[line-codeboxStartLine] = lastNonSpaceIndex - int(lineMap[line]) + codeboxStart
\t\t\t\tlastNonSpaceIndex = -1
\t\t\t\tline++
\t\t\t}
\t\t\tcontinue
\t\t}
\t\tif !lineIndentCalculated && !unicode.IsSpace(char) {
\t\t\tlineIndentCalculated = true
\t\t\tlineStarts[line-codeboxStartLine] = i - int(lineMap[line]) + codeboxStart
\t\t\tindentSize = min(indentSize, lineStarts[line-codeboxStartLine])
\t\t}
\t\tif lineIndentCalculated && !unicode.IsSpace(char) {
\t\t\tlastNonSpaceIndex = i + 1
\t\t}
\t}
\tif line == codeboxEndLine {
\t\tlineEnds[line-codeboxStartLine] = lastNonSpaceIndex - int(lineMap[line]) + codeboxStart
\t}
\tdiagnosticHighlightActive := false
\tlastLineNumber := strconv.Itoa(codeboxEndLine + 1)
\tfor line := codeboxStartLine; line <= codeboxEndLine; line++ {
\t\tw.WriteString("  \\x1b[2m│ ")
\t\tif line == codeboxEndLine {
\t\t\tw.WriteString(lastLineNumber)
\t\t} else {
\t\t\tnumber := strconv.Itoa(line + 1)
\t\t\tif len(number) < len(lastLineNumber) {
\t\t\t\tw.WriteByte(' ')
\t\t\t}
\t\t\tw.WriteString(number)
\t\t}
\t\tw.WriteString(" │\\x1b[0m  ")
\t\tlineTextStart := int(lineMap[line]) + indentSize
\t\tunderlineStart := max(lineTextStart, int(lineMap[line])+lineStarts[line-codeboxStartLine])
\t\tunderlineEnd := underlineStart
\t\tlineTextEnd := max(int(lineMap[line])+lineEnds[line-codeboxStartLine], lineTextStart)
\t\tif diagnosticHighlightActive {
\t\t\tunderlineEnd = lineTextEnd
\t\t} else if int(lineMap[line]) <= diagnosticStart && (line == len(lineMap) || diagnosticStart < int(lineMap[line+1])) {
\t\t\tunderlineStart = min(max(lineTextStart, diagnosticStart), lineTextEnd)
\t\t\tunderlineEnd = lineTextEnd
\t\t\tdiagnosticHighlightActive = true
\t\t}
\t\tif int(lineMap[line]) <= diagnosticEnd && (line == len(lineMap) || diagnosticEnd < int(lineMap[line+1])) {
\t\t\tunderlineEnd = min(max(underlineStart, diagnosticEnd), lineTextEnd)
\t\t\tdiagnosticHighlightActive = false
\t\t}
\t\tif underlineStart != underlineEnd {
\t\t\tw.WriteString(text[lineTextStart:underlineStart])
\t\t\tw.Write([]byte{
\t\t\t\t0x1b, '[', '4', 'm',
\t\t\t\t0x1b, '[', '4', ':', '3', 'm',
\t\t\t\t0x1b, '[', '5', '8', ':', '5', ':', '1', '6', '0', 'm',
\t\t\t\t0x1b, '[', '3', '8', ';', '5', ';', '1', '6', '0', 'm',
\t\t\t\t0x1b, '[', '2', '2', ';', '4', '9', 'm',
\t\t\t})
\t\t\tw.WriteString(text[underlineStart:underlineEnd])
\t\t\tw.Write([]byte{0x1b, '[', '0', 'm'})
\t\t\tw.WriteString(text[underlineEnd:lineTextEnd])
\t\t} else if lineTextStart != lineTextEnd {
\t\t\tw.WriteString(text[lineTextStart:lineTextEnd])
\t\t}
\t\tw.WriteByte('\\n')
\t}
\tw.WriteString("  \\x1b[2m╰────────────────────────────────\\x1b[0m\\n\\n")
}

const usage = \` lintcn - type-aware TypeScript linter with custom rules

Usage:
    lintcn-binary [OPTIONS]

Options:
    --tsconfig PATH   Which tsconfig to use. Defaults to tsconfig.json.
    --list-files      List matched files
    -h, --help        Show help
\`

func runMain() int {
\tflag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
\tvar (
\t\thelp      bool
\t\ttsconfig  string
\t\tlistFiles bool
\t\ttraceOut       string
\t\tcpuprofOut     string
\t\tsingleThreaded bool
\t)
\tflag.StringVar(&tsconfig, "tsconfig", "", "which tsconfig to use")
\tflag.BoolVar(&listFiles, "list-files", false, "list matched files")
\tflag.BoolVar(&help, "help", false, "show help")
\tflag.BoolVar(&help, "h", false, "show help")
\tflag.StringVar(&traceOut, "trace", "", "file to put trace to")
\tflag.StringVar(&cpuprofOut, "cpuprof", "", "file to put cpu profiling to")
\tflag.BoolVar(&singleThreaded, "singleThreaded", false, "run in single threaded mode")
\tflag.Parse()
\tif help {
\t\tflag.Usage()
\t\treturn 0
\t}
\ttimeBefore := time.Now()
\tif done, err := recordTrace(traceOut); err != nil {
\t\tos.Stderr.WriteString(err.Error())
\t\treturn 1
\t} else {
\t\tdefer done()
\t}
\tif done, err := recordCpuprof(cpuprofOut); err != nil {
\t\tos.Stderr.WriteString(err.Error())
\t\treturn 1
\t} else {
\t\tdefer done()
\t}
\tcurrentDirectory, err := os.Getwd()
\tif err != nil {
\t\tfmt.Fprintf(os.Stderr, "error getting current directory: %v\\n", err)
\t\treturn 1
\t}
\tcurrentDirectory = tspath.NormalizePath(currentDirectory)
\tfs := bundled.WrapFS(cachedvfs.From(osvfs.FS()))
\tvar configFileName string
\tif tsconfig == "" {
\t\tconfigFileName = tspath.ResolvePath(currentDirectory, "tsconfig.json")
\t\tif !fs.FileExists(configFileName) {
\t\t\tfs = utils.NewOverlayVFS(fs, map[string]string{
\t\t\t\tconfigFileName: "{}",
\t\t\t})
\t\t}
\t} else {
\t\tconfigFileName = tspath.ResolvePath(currentDirectory, tsconfig)
\t\tif !fs.FileExists(configFileName) {
\t\t\tfmt.Fprintf(os.Stderr, "error: tsconfig %q doesn't exist", tsconfig)
\t\t\treturn 1
\t\t}
\t}
\tcurrentDirectory = tspath.GetDirectoryPath(configFileName)
\thost := utils.CreateCompilerHost(currentDirectory, fs)
\tcomparePathOptions := tspath.ComparePathsOptions{
\t\tCurrentDirectory:          host.GetCurrentDirectory(),
\t\tUseCaseSensitiveFileNames: host.FS().UseCaseSensitiveFileNames(),
\t}
\tprogram, _, err := utils.CreateProgram(singleThreaded, fs, currentDirectory, configFileName, host, false)
\tif err != nil {
\t\tfmt.Fprintf(os.Stderr, "error creating TS program: %v", err)
\t\treturn 1
\t}
\tif program == nil {
\t\tfmt.Fprintf(os.Stderr, "error creating TS program")
\t\treturn 1
\t}
\tfiles := []*ast.SourceFile{}
\tcwdPath := string(tspath.ToPath("", currentDirectory, program.Host().FS().UseCaseSensitiveFileNames()).EnsureTrailingDirectorySeparator())
\tvar matchedFiles strings.Builder
\tfor _, file := range program.SourceFiles() {
\t\tp := string(file.Path())
\t\tif strings.Contains(p, "/node_modules/") {
\t\t\tcontinue
\t\t}
\t\tif fileName, matched := strings.CutPrefix(p, cwdPath); matched {
\t\t\tif listFiles {
\t\t\t\tmatchedFiles.WriteString("Found file: ")
\t\t\t\tmatchedFiles.WriteString(fileName)
\t\t\t\tmatchedFiles.WriteByte('\\n')
\t\t\t}
\t\t\tfiles = append(files, file)
\t\t}
\t}
\tif listFiles {
\t\tos.Stdout.WriteString(matchedFiles.String())
\t}
\tslices.SortFunc(files, func(a *ast.SourceFile, b *ast.SourceFile) int {
\t\treturn len(b.Text()) - len(a.Text())
\t})
\tvar wg sync.WaitGroup
\tdiagnosticsChan := make(chan rule.RuleDiagnostic, 4096)
\terrorsCount := 0
\twg.Go(func() {
\t\tw := bufio.NewWriterSize(os.Stdout, 4096*100)
\t\tdefer w.Flush()
\t\tfor d := range diagnosticsChan {
\t\t\terrorsCount++
\t\t\tif errorsCount == 1 {
\t\t\t\tw.WriteByte('\\n')
\t\t\t}
\t\t\tprintDiagnostic(d, w, comparePathOptions)
\t\t\tif w.Available() < 4096 {
\t\t\t\tw.Flush()
\t\t\t}
\t\t}
\t})
\terr = linter.RunLinterOnProgram(
\t\tutils.GetLogLevel(),
\t\tprogram,
\t\tfiles,
\t\truntime.GOMAXPROCS(0),
\t\tfunc(sourceFile *ast.SourceFile) []linter.ConfiguredRule {
\t\t\treturn utils.Map(allRules, func(r rule.Rule) linter.ConfiguredRule {
\t\t\t\treturn linter.ConfiguredRule{
\t\t\t\t\tName: r.Name,
\t\t\t\t\tRun: func(ctx rule.RuleContext) rule.RuleListeners {
\t\t\t\t\t\treturn r.Run(ctx, nil)
\t\t\t\t\t},
\t\t\t\t}
\t\t\t})
\t\t},
\t\tfunc(d rule.RuleDiagnostic) {
\t\t\tdiagnosticsChan <- d
\t\t},
\t\tfunc(d diagnostic.Internal) {},
\t\tlinter.Fixes{
\t\t\tFix:            true,
\t\t\tFixSuggestions: true,
\t\t},
\t\tlinter.TypeErrors{
\t\t\tReportSyntactic: false,
\t\t\tReportSemantic:  false,
\t\t},
\t)
\tclose(diagnosticsChan)
\tif err != nil {
\t\tfmt.Fprintf(os.Stderr, "error running linter: %v\\n", err)
\t\treturn 1
\t}
\twg.Wait()
\terrorsColor := "\\x1b[1m"
\tif errorsCount == 0 {
\t\terrorsColor = "\\x1b[1;32m"
\t}
\terrorsText := "errors"
\tif errorsCount == 1 {
\t\terrorsText = "error"
\t}
\tfilesText := "files"
\tif len(files) == 1 {
\t\tfilesText = "file"
\t}
\trulesText := "rules"
\tif len(allRules) == 1 {
\t\trulesText = "rule"
\t}
\tthreadsCount := 1
\tif !singleThreaded {
\t\tthreadsCount = runtime.GOMAXPROCS(0)
\t}
\tfmt.Fprintf(
\t\tos.Stdout,
\t\t"Found %v%v\\x1b[0m %v \\x1b[2m(linted \\x1b[1m%v\\x1b[22m\\x1b[2m %v with \\x1b[1m%v\\x1b[22m\\x1b[2m %v in \\x1b[1m%v\\x1b[22m\\x1b[2m using \\x1b[1m%v\\x1b[22m\\x1b[2m threads)\\n",
\t\terrorsColor,
\t\terrorsCount,
\t\terrorsText,
\t\tlen(files),
\t\tfilesText,
\t\tlen(allRules),
\t\trulesText,
\t\ttime.Since(timeBefore).Round(time.Millisecond),
\t\tthreadsCount,
\t)
\treturn 0
}

func main() {
\tos.Exit(runMain())
}
`
}
