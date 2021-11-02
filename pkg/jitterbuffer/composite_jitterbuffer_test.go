package jitterbuffer

import (
	"testing"

	"go.uber.org/goleak"
)

func TestCompositeJitterBuffer_Simple(t *testing.T) {

	goleak.VerifyNone(t)
}
