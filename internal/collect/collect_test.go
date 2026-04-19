package collect

import (
	"testing"

	healthv1alpha1 "github.com/k8s-health-ai/k8s-health-ai/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildResourceUsage_Container(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			},
		},
	}
	ru := BuildResourceUsage(pod, "app")
	if ru == nil {
		t.Fatal("expected ResourceUsage")
	}
	want := &healthv1alpha1.ResourceUsage{
		CpuRequest:    "100m",
		CpuLimit:      "500m",
		MemoryRequest: "128Mi",
		MemoryLimit:   "512Mi",
	}
	if *ru != *want {
		t.Fatalf("got %+v want %+v", ru, want)
	}
}

func TestBuildResourceUsage_InitContainer(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name: "init",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("50m"),
						},
					},
				},
			},
		},
	}
	ru := BuildResourceUsage(pod, "init")
	if ru == nil || ru.CpuRequest != "50m" {
		t.Fatalf("got %+v", ru)
	}
}

func TestBuildResourceUsage_None(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
	}
	if BuildResourceUsage(pod, "app") != nil {
		t.Fatal("expected nil")
	}
	if BuildResourceUsage(nil, "x") != nil {
		t.Fatal("expected nil")
	}
}
