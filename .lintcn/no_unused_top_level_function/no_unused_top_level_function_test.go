package no_unused_top_level_function

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func isolatedTSConfig(fileNames ...string) string {
	filesJSON, err := json.Marshal(fileNames)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(`{
  "compilerOptions": {
    "jsx": "preserve",
    "target": "esnext",
    "module": "commonjs",
    "strict": true,
    "esModuleInterop": true,
    "lib": ["esnext"],
    "experimentalDecorators": true,
    "skipLibCheck": true,
    "skipDefaultLibCheck": true,
    "types": []
  },
  "files": %s
}`, filesJSON)
}

func testFiles(fileName string, files map[string]string) (string, map[string]string) {
	tsconfigFileName := fileName + ".tsconfig.json"
	fileNames := make([]string, 0, len(files)+1)
	fileNames = append(fileNames, fileName)
	for extraFileName := range files {
		fileNames = append(fileNames, extraFileName)
	}
	slices.Sort(fileNames)

	withTSConfig := make(map[string]string, len(files)+1)
	for extraFileName, source := range files {
		withTSConfig[extraFileName] = source
	}
	withTSConfig[tsconfigFileName] = isolatedTSConfig(fileNames...)

	return tsconfigFileName, withTSConfig
}

func validCase(fileName string, code string, files map[string]string) rule_tester.ValidTestCase {
	tsconfigFileName, testFiles := testFiles(fileName, files)
	return rule_tester.ValidTestCase{
		Code:     code,
		FileName: fileName,
		TSConfig: tsconfigFileName,
		Files:    testFiles,
	}
}

func invalidCase(fileName string, code string, files map[string]string, errors []rule_tester.InvalidTestCaseError) rule_tester.InvalidTestCase {
	tsconfigFileName, testFiles := testFiles(fileName, files)
	return rule_tester.InvalidTestCase{
		Code:     code,
		FileName: fileName,
		TSConfig: tsconfigFileName,
		Files:    testFiles,
		Errors:   errors,
	}
}

func TestNoUnusedTopLevelFunction(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(
		fixtures.GetRootDir(),
		"tsconfig.minimal.json",
		t,
		&NoUnusedTopLevelFunctionRule,
		validCases,
		invalidCases,
	)
}

var validCases = []rule_tester.ValidTestCase{
	validCase("used-function.ts", `function used() { return 1 } const value = used();`, nil),
	validCase("exported-function.ts", `export function publicApi() { return 1 }`, nil),
	validCase("shared-function.ts", `function shared() { return 1 } export { shared }`, map[string]string{
		"shared-consumer.ts": `import { shared } from "./shared-function"; const value = shared();`,
	}),
	validCase("arrow-helper.ts", `const helper = () => 1;`, nil),
	validCase("nested-function.ts", `function outer() { function inner() { return 1 } return inner() } const value = outer()`, nil),
	validCase("callback-function.ts", `function callback() { return 1 } export { callback }`, map[string]string{
		"callback-consumer.ts": `import { callback } from "./callback-function"; const handlers = [callback]; handlers[0]();`,
	}),
}

var invalidCases = []rule_tester.InvalidTestCase{
	invalidCase("orphan-function.ts", `function orphan() { return 1 }`, nil, []rule_tester.InvalidTestCaseError{
		{MessageId: "unusedTopLevelFunction"},
	}),
	invalidCase("unused-helper.ts", `function helper() { return 1 }`, map[string]string{
		"unused-helper-consumer.ts": `const value = 1;`,
	}, []rule_tester.InvalidTestCaseError{
		{MessageId: "unusedTopLevelFunction"},
	}),
	invalidCase("recursive-function.ts", `function loop(value: number): number { if (value <= 0) return 0; return loop(value - 1) }`, nil, []rule_tester.InvalidTestCaseError{
		{MessageId: "unusedTopLevelFunction"},
	}),
}
