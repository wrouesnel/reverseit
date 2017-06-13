package main

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MainSuite struct{}

var _ = Suite(&MainSuite{})

// TestNothing is a dummy test for the time being.
func (s *MainSuite) TestNothing(c *C) {
	c.Assert(true, Equals, true)
}
