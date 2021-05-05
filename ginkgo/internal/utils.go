package internal

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/onsi/ginkgo/formatter"
	"github.com/onsi/ginkgo/ginkgo/command"
)

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func GoFmt(path string) {
	out, err := exec.Command("go", "fmt", path).CombinedOutput()
	if err != nil {
		command.AbortIfError(fmt.Sprintf("Could not fmt:\n%s\n", string(out)), err)
	}
}

func PluralizedWord(singular, plural string, count int) string {
	if count == 1 {
		return singular
	}
	return plural
}

func FailedSuitesReport(suites TestSuites, f formatter.Formatter) string {
	out := ""
	out += "There were failures detected in the following suites:\n"

	maxPackageNameLength := 0
	for _, suite := range suites.WithState(TestSuiteStateFailureStates...) {
		if len(suite.PackageName) > maxPackageNameLength {
			maxPackageNameLength = len(suite.PackageName)
		}
	}

	packageNameFormatter := fmt.Sprintf("%%%ds", maxPackageNameLength)
	for _, suite := range suites {
		switch suite.State {
		case TestSuiteStateFailed:
			out += f.Fi(1, "{{red}}"+packageNameFormatter+" {{gray}}%s{{/}}\n", suite.PackageName, suite.Path)
		case TestSuiteStateFailedToCompile:
			out += f.Fi(1, "{{red}}"+packageNameFormatter+" {{gray}}%s {{magenta}}[Compilation failure]{{/}}\n", suite.PackageName, suite.Path)
		case TestSuiteStateFailedDueToTimeout:
			out += f.Fi(1, "{{red}}"+packageNameFormatter+" {{gray}}%s {{orange}}[%s]{{/}}\n", suite.PackageName, suite.Path, TIMEOUT_ELAPSED_FAILURE_REASON)
		}
	}
	return out
}