package resp_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/bsm/redeo/resp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// --------------------------------------------------------------------

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "resp")
}

// --------------------------------------------------------------------

func MatchCommand(expected ...string) types.GomegaMatcher {
	return &commandMatcher{expected: expected}
}

type commandMatcher struct {
	expected []string
}

func (m *commandMatcher) Match(actual interface{}) (bool, error) {
	cmd, ok := actual.(*resp.Command)
	if !ok {
		return false, fmt.Errorf("MatchCommand matcher expects a Command, but was %T", actual)
	}
	return reflect.DeepEqual(m.expected, cmdToSlice(cmd)), nil
}

func (m *commandMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%#v\nto match\n\t%#v", actual, m.expected)
}
func (m *commandMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%#v\nnot to match\n\t%#v", actual, m.expected)
}

func cmdToSlice(cmd *resp.Command) []string {
	res := []string{cmd.Name}
	for _, arg := range cmd.Args() {
		res = append(res, string(arg))
	}
	return res
}
