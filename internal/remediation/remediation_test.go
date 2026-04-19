package remediation

import (
	"testing"

	"github.com/k8s-health-ai/k8s-health-ai/internal/detect"
)

func TestForFailure_KnownTypes(t *testing.T) {
	for _, ft := range []detect.FailureType{
		detect.OOMKilled,
		detect.CrashLoopBackOff,
		detect.ImagePullBackOff,
		detect.InitContainerError,
		detect.PendingScheduling,
	} {
		h := ForFailure(ft)
		if len(h) < 2 {
			t.Fatalf("%s: expected multiple hints, got %v", ft, h)
		}
	}
}

func TestForFailure_UnknownUsesDefault(t *testing.T) {
	h := ForFailure(detect.FailureType("UnknownMadeUp"))
	if len(h) < 1 {
		t.Fatal("expected default hints")
	}
}
