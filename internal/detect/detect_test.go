package detect

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestClassify_OOM(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "app",
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"},
					},
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
					},
				},
			},
		},
	}
	ft, c, ok := Classify(pod)
	if !ok || ft != OOMKilled || c != "app" {
		t.Fatalf("got %v %q %v", ft, c, ok)
	}
}

func TestClassify_ImagePull(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"},
				}},
			},
		},
	}
	ft, c, ok := Classify(pod)
	if !ok || ft != ImagePullBackOff || c != "app" {
		t.Fatalf("got %v %q %v", ft, c, ok)
	}
}

func TestClassify_CrashLoop(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
				}},
			},
		},
	}
	ft, c, ok := Classify(pod)
	if !ok || ft != CrashLoopBackOff || c != "app" {
		t.Fatalf("got %v %q %v", ft, c, ok)
	}
}

func TestDiagnosisNameStable(t *testing.T) {
	a := DiagnosisName("ns", "pod", CrashLoopBackOff)
	b := DiagnosisName("ns", "pod", CrashLoopBackOff)
	if a != b {
		t.Fatal("name not stable")
	}
	if DiagnosisName("ns", "pod2", CrashLoopBackOff) == a {
		t.Fatal("expected different name for different pod")
	}
}
