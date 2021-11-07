package test

import (
	"testing"

	"go.uber.org/goleak"
)

func TestNetSim_Simple(t *testing.T) {
	goleak.VerifyNone(t)
}