package chef_load

import (
	"testing"

	"strings"

	"github.com/stretchr/testify/assert"
)

func TestGenerateNodeName(t *testing.T) {
	nodeName := generateNodeName("prefix")

	nodeNameTokenized := strings.Split(nodeName, "-")
	assert.Len(t, nodeNameTokenized, 4, "")
}
