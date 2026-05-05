package api_test

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// startLimiterCleanup uses a 5-minute ticker and has no shutdown path yet (D2).
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("github.com/anIcedAntFA/goshort/internal/api.startLimiterCleanup.func1"),
	)
}
