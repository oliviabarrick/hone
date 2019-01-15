package job

import (
	"testing"
	"github.com/zclconf/go-cty/cty"
	"github.com/stretchr/testify/assert"
)

func TestSetMapString(t *testing.T) {
	objMap := map[string]cty.Value{}
	j := Job{}

	j.setMapString(objMap, "test", nil)

	assert.Equal(t, true, objMap["test"].IsNull())
}
