//go:build system_tests

package system_tests

import (
	"actionforge/graph-runner/cmd"
	"actionforge/graph-runner/utils"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"

	// initialize all nodes
	_ "actionforge/graph-runner/nodes"
)

// Test run node and environment variables
// array using an incoming connection
func Test_Simple(t *testing.T) {
	defer utils.LoggerString.Clear()

	// Test the run node
	exitCode, err := runGraphFile("system_tests/test_simple.yml")
	if err != nil {
		t.Fatal(err)
	} else if exitCode != 0 {
		t.Fatal("exitCode != 0")
	}

	actual := utils.LoggerString.String()

	expectedString := `🟢 Execute 'Start (start)'
🟢 Execute 'Run (run-v1-koala-giraffe-cranberry)'
Hello World!
🟢 Execute 'Run (run-v1-purple-dog-koala)'
Success
`

	if !diffStrings(actual, expectedString) {
		t.Fatal()
	}
}

// Like Test_Simple but also tests environment variables
// directly inside a group input.
func Test_Simple2(t *testing.T) {
	defer utils.LoggerString.Clear()

	// Test the run node, env node, and string format node.
	exitCode, err := runGraphFile("system_tests/test_simple2.yml")
	if err != nil {
		t.Fatal(err)
	} else if exitCode != 0 {
		t.Fatal("exitCode != 0")
	}

	actual := utils.LoggerString.String()

	expectedString := `🟢 Execute 'Start (start)'
Execute 'Run (run-v1-yellow-squirrel-octopus)'
World
🟢 Execute 'Run (run-v1-orange-squirrel-koala)'
Hello World!
🟢 Execute 'Run (run-v1-koala-lemon-cranberry)'
Hello 1234!
`

	if !diffStrings(actual, expectedString) {
		t.Fatal()
	}
}

// Like Test_Simple2 but also tests environment variables
// passed via SetEnv.
func Test_Simple3(t *testing.T) {
	defer utils.LoggerString.Clear()

	t.Setenv("BAS", "Universe")

	// Test the run node, env node, and string format node.
	exitCode, err := runGraphFile("system_tests/test_simple3.yml")
	if err != nil {
		t.Fatal(err)
	} else if exitCode != 0 {
		t.Fatal("exitCode != 0")
	}

	actual := utils.LoggerString.String()

	expectedString := `🟢 Execute 'Start (start)'
🟢 Execute 'Run (run-v1-yellow-squirrel-octopus)'
World
🟢 Execute 'Run (run-v1-orange-squirrel-koala)'
Hello World!
🟢 Execute 'Run (run-v1-koala-lemon-cranberry)'
Hello 1234!
🟢 Execute 'Run (run-v1-apple-zebra-squirrel)'
Hello Universe!
`

	if !diffStrings(actual, expectedString) {
		t.Fatal()
	}
}

// Test several for comparisions and conditions
func Test_IfAndVariousCompare(t *testing.T) {
	defer utils.LoggerString.Clear()

	testCase := map[string]string{
		"Hello World!": `🟢 Execute 'Start (start)'
🟢 Execute 'Branch (if-v1-koala-peach-gray)'
🟢 Execute 'Run (run-v1-penguin-pineapple-pineapple)'
Yes
`,
		"Hello Universe!": `🟢 Execute 'Start (start)'
🟢 Execute 'Branch (if-v1-koala-peach-gray)'
🟢 Execute 'Run (run-v1-mango-silver-silver)'
No
`,
	}

	for env, expectedString := range testCase {

		// Clear the logger string from the previous test.
		utils.LoggerString.Clear()

		t.Setenv("FOO", env)

		// Test the run node, env node, and string format node.
		exitCode, err := runGraphFile("system_tests/test_if.yml")
		if err != nil {
			t.Fatal(err)
		} else if exitCode != 0 {
			t.Fatal("exitCode != 0")
		}

		actual := utils.LoggerString.String()

		if !diffStrings(actual, expectedString) {
			t.Fatal()
		}
	}
}

// Simple test of the for loop
func Test_For(t *testing.T) {
	defer utils.LoggerString.Clear()

	// Test the run node, env node, and string format node.
	exitCode, err := runGraphFile("system_tests/test_for.yml")
	if err != nil {
		t.Fatal(err)
	} else if exitCode != 0 {
		t.Fatal("exitCode != 0")
	}

	actual := utils.LoggerString.String()

	expectedString := `🟢 Execute 'Start (start)'
🟢 Execute 'For (for-v1-snake-strawberry-tiger)'
🟢 Execute 'Run (run-v1-butterfly-gray-shark)'
3
🟢 Execute 'Run (run-v1-butterfly-gray-shark)'
4
🟢 Execute 'Run (run-v1-butterfly-gray-shark)'
5
🟢 Execute 'Run (run-v1-butterfly-gray-shark)'
6
🟢 Execute 'Run (run-v1-butterfly-gray-shark)'
7
🟢 Execute 'Run (run-v1-cherry-banana-brown)'
Done
`
	if !diffStrings(actual, expectedString) {
		t.Fatal()
	}
}

// Simple test to check the boolean nodes
func Test_Bool(t *testing.T) {
	defer utils.LoggerString.Clear()

	// Test the run node, env node, and string format node.
	exitCode, err := runGraphFile("system_tests/test_bool.yml")
	if err != nil {
		t.Fatal(err)
	} else if exitCode != 0 {
		t.Fatal("exitCode != 0")
	}

	actual := utils.LoggerString.String()

	expectedString := `🟢 Execute 'Start (start)'
🟢 Execute 'Run (run-v1-giraffe-dolphin-pink)'
AND 0&&0=false 1&&0=false 0&&1=false 1&&1=true
OR 0&&0=false 1&&0=true 0&&1=true 1&&1=true
XOR 0&&0=false 1&&0=true 0&&1=true 1&&1=false
XAND 0&&0=true 1&&0=false 0&&1=false 1&&1=true
`

	if !diffStrings(actual, expectedString) {
		t.Fatal()
	}
}

// Simple test to check that option inputs can
// be controlled by strings and number outputs
func Test_Option(t *testing.T) {
	defer utils.LoggerString.Clear()

	// Test the run node, env node, and string format node.
	exitCode, err := runGraphFile("system_tests/test_option.yml")
	if err != nil {
		t.Fatal(err)
	} else if exitCode != 0 {
		t.Fatal("exitCode != 0")
	}

	actual := utils.LoggerString.String()

	expectedString := `🟢 Execute 'Start (start)'
🟢 Execute 'Run (run-v1-cranberry-cranberry-grape)'
python
🟢 Execute 'Run (run-v1-parrot-kiwi-gold)'
Hello World!
`

	if !diffStrings(actual, expectedString) {
		t.Fatal()
	}
}

// Simple test of the parallel-exec and wait-for node.
func Test_Parallel(t *testing.T) {
	defer utils.LoggerString.Clear()

	// Test the run node, env node, and string format node.
	exitCode, err := runGraphFile("system_tests/test_parallel.yml")
	if err != nil {
		t.Fatal(err)
	} else if exitCode != 0 {
		t.Fatal("exitCode != 0")
	}

	actual := utils.LoggerString.String()

	// Due to the nature of this parallel test,
	// the order of the result is not guaranteed.
	// Extract all output, sort it and then compare it.
	r := regexp.MustCompile(`Goroutine (\w+)`)
	rs := r.FindAllStringSubmatch(actual, -1)
	if len(rs) != 6 {
		t.Fatalf("Unexpected result from parallel test:\n%v", actual)
	}

	results := make([]string, 0)
	for _, r := range rs {
		results = append(results, r[1])
	}

	sort.Strings(results)
	actual = strings.Join(results, " ")

	expectedString := `1 2 3 4 5 Done`

	if !diffStrings(actual, expectedString) {
		t.Fatal()
	}
}

func runGraphFile(graphFileName string) (exitCode int, err error) {
	root := utils.FindProjectRoot()
	graphFile := filepath.Join(root, graphFileName)

	exitCode, err = cmd.ExecuteRun(graphFile)
	if err != nil {
		return exitCode, err
	} else if exitCode != 0 {
		return exitCode, nil
	}

	return 0, nil
}

func diffStrings(actual string, expected string) bool {
	dmp := diffmatchpatch.New()

	actual = utils.NormalizeLineEndings(actual)
	expected = utils.NormalizeLineEndings(expected)

	diffs := dmp.DiffMain(actual, expected, false)

	different := false
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			different = true
			break
		}
	}
	if different {
		fmt.Println("\n\n-----------\nExpected output vs actual output (inline-diff):")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
	return !different
}

func init() {
	utils.EnableStringLogging(true)
}
