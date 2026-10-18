package identifier

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIdentifier(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Identifier Suite")
}
