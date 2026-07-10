package basic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddOne(t *testing.T) {
	// var (
	// 	input  = 1
	// 	output = 3
	// )

	// actual := AddOne(input)

	// if actual != output {
	// 	t.Errorf("Actual: %d, Input: %d, Output: %d", actual, input, output)
	// }

	// fmt.Printf("Actual: %d, Input: %d, Output: %d", actual, input, output)

	// assert.Equal(t, AddOne(1), 3, "AddOne(1) should be 2")
	// assert.NotEqual(t, 2, 3)
	// assert.Nil(t, nil, nil)

	assert.Equal(t, AddOne2(1), 3, "AddOne(1) should be 2")
	assert.NotEqual(t, 2, 3)
	assert.Nil(t, nil, nil)
}

func TestRequire(t *testing.T) {
	require.Equal(t, 2, 3)
	fmt.Println("Not execute")
}

func TestAsssert(t *testing.T) {
	assert.Equal(t, 2, 3)
	fmt.Println("Execute")
}
