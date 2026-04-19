package detect

import (
	"strings"
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

func TestClassify_InitContainerError_TerminatedNonZero(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "init-fetch",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "Error"},
					},
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{Reason: "PodInitializing"},
				}},
			},
		},
	}
	ft, c, ok := Classify(pod)
	if !ok || ft != InitContainerError || c != "init-fetch" {
		t.Fatalf("got %v %q %v", ft, c, ok)
	}
}

func TestClassify_InitContainerError_WaitingReasonError(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "init-migrate",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CreateContainerConfigError", Message: "secret missing"},
					},
				},
			},
		},
	}
	ft, c, ok := Classify(pod)
	if !ok || ft != InitContainerError || c != "init-migrate" {
		t.Fatalf("got %v %q %v", ft, c, ok)
	}
}

func TestClassify_PendingScheduling_FailedScheduling(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodScheduled,
					Status: corev1.ConditionFalse,
					Reason: "FailedScheduling",
				},
			},
		},
	}
	ft, c, ok := Classify(pod)
	if !ok || ft != PendingScheduling || c != "" {
		t.Fatalf("got %v %q %v", ft, c, ok)
	}
}

func TestClassify_PendingScheduling_CreateContainerConfigError(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "app",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CreateContainerConfigError"},
					},
				},
			},
		},
	}
	ft, c, ok := Classify(pod)
	if !ok || ft != PendingScheduling || c != "app" {
		t.Fatalf("got %v %q %v", ft, c, ok)
	}
}

func TestFingerprint_InitContainer(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "init",
					RestartCount: 0,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "ErrImagePull", Message: "pull failed"},
					},
				},
			},
		},
	}
	fp := Fingerprint(pod, "init")
	if fp == "" || !strings.Contains(fp, "init|") || !strings.Contains(fp, "|W:ErrImagePull") {
		t.Fatalf("unexpected fingerprint: %q", fp)
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
