package gors

import (
	"testing"
	"libs/assrt"
)

func TestScopes(t *testing.T) {
	assert := assrt.NewAssert(t)
	assert.Equal([]Scope{Scope{"name1", true}, Scope{"name2", false}}, parseScopes("name1:rw name2:r"))
}
