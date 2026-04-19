package controller

import (
	"context"
	"testing"

	healthv1alpha1 "github.com/k8s-health-ai/k8s-health-ai/api/v1alpha1"
	"github.com/k8s-health-ai/k8s-health-ai/internal/detect"
	"github.com/k8s-health-ai/k8s-health-ai/internal/llm"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = healthv1alpha1.AddToScheme(scheme)
	return scheme
}

type mockProvider struct {
	diagnosis llm.Diagnosis
	err       error
}

func (m *mockProvider) Diagnose(ctx context.Context, fc llm.FailureContext) (llm.Diagnosis, error) {
	return m.diagnosis, m.err
}

func TestReconcile_CreatesClusterDiagnosis(t *testing.T) {
	scheme := newTestScheme()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crash-pod",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "myapp:v1"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "app",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "CrashLoopBackOff",
							Message: "back-off 5m0s restarting failed container",
						},
					},
					RestartCount: 5,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		WithStatusSubresource(&healthv1alpha1.ClusterDiagnosis{}).
		Build()

	fakeK8s := kubefake.NewSimpleClientset(pod)

	mockLLM := &mockProvider{
		diagnosis: llm.Diagnosis{
			RootCause:      "Application crashing due to missing config",
			Severity:       "high",
			RecommendedFix: "Add required environment variables",
			Model:          "test-model",
		},
	}

	r := &PodReconciler{
		Client: fakeClient,
		K8s:    fakeK8s,
		Scheme: scheme,
		LLM:    mockLLM,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "crash-pod",
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	expectedName := detect.DiagnosisName("default", "crash-pod", detect.CrashLoopBackOff)
	var cd healthv1alpha1.ClusterDiagnosis
	err = fakeClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      expectedName,
	}, &cd)
	if err != nil {
		t.Fatalf("Failed to get ClusterDiagnosis: %v", err)
	}

	if cd.Spec.TargetRef.Kind != "Pod" {
		t.Errorf("Expected TargetRef.Kind=Pod, got %s", cd.Spec.TargetRef.Kind)
	}
	if cd.Spec.TargetRef.Name != "crash-pod" {
		t.Errorf("Expected TargetRef.Name=crash-pod, got %s", cd.Spec.TargetRef.Name)
	}
	if cd.Spec.TargetRef.Namespace != "default" {
		t.Errorf("Expected TargetRef.Namespace=default, got %s", cd.Spec.TargetRef.Namespace)
	}
	if cd.Spec.FailureType != string(detect.CrashLoopBackOff) {
		t.Errorf("Expected FailureType=CrashLoopBackOff, got %s", cd.Spec.FailureType)
	}
}

func TestReconcile_SkipsHealthyPod(t *testing.T) {
	scheme := newTestScheme()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "healthy-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "myapp:v1"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	fakeK8s := kubefake.NewSimpleClientset(pod)

	r := &PodReconciler{
		Client: fakeClient,
		K8s:    fakeK8s,
		Scheme: scheme,
		LLM:    &mockProvider{},
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "healthy-pod",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if result.Requeue || result.RequeueAfter != 0 {
		t.Errorf("Expected no requeue for healthy pod, got %+v", result)
	}

	var cdList healthv1alpha1.ClusterDiagnosisList
	err = fakeClient.List(context.Background(), &cdList, client.InNamespace("default"))
	if err != nil {
		t.Fatalf("Failed to list ClusterDiagnosis: %v", err)
	}
	if len(cdList.Items) != 0 {
		t.Errorf("Expected no ClusterDiagnosis for healthy pod, got %d", len(cdList.Items))
	}
}

func TestReconcile_UpdatesStatusOnLLMSuccess(t *testing.T) {
	scheme := newTestScheme()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oom-pod",
			Namespace: "test-ns",
			UID:       "oom-uid-456",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "worker", Image: "worker:v2"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "worker",
					RestartCount: 3,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "OOMKilled",
							ExitCode: 137,
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		WithStatusSubresource(&healthv1alpha1.ClusterDiagnosis{}).
		Build()

	fakeK8s := kubefake.NewSimpleClientset(pod)

	mockLLM := &mockProvider{
		diagnosis: llm.Diagnosis{
			RootCause:      "Container exceeded memory limits",
			Severity:       "critical",
			RecommendedFix: "Increase memory limit to 512Mi",
			Model:          "gpt-4",
		},
	}

	r := &PodReconciler{
		Client: fakeClient,
		K8s:    fakeK8s,
		Scheme: scheme,
		LLM:    mockLLM,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-ns",
			Name:      "oom-pod",
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	expectedName := detect.DiagnosisName("test-ns", "oom-pod", detect.OOMKilled)
	var cd healthv1alpha1.ClusterDiagnosis
	err = fakeClient.Get(context.Background(), client.ObjectKey{
		Namespace: "test-ns",
		Name:      expectedName,
	}, &cd)
	if err != nil {
		t.Fatalf("Failed to get ClusterDiagnosis: %v", err)
	}

	if cd.Status.Phase != healthv1alpha1.PhaseReady {
		t.Errorf("Expected Phase=Ready, got %s", cd.Status.Phase)
	}
	if cd.Status.RootCause != "Container exceeded memory limits" {
		t.Errorf("Expected RootCause='Container exceeded memory limits', got %s", cd.Status.RootCause)
	}
	if cd.Status.Severity != "critical" {
		t.Errorf("Expected Severity=critical, got %s", cd.Status.Severity)
	}
	if cd.Status.RecommendedFix != "Increase memory limit to 512Mi" {
		t.Errorf("Expected RecommendedFix='Increase memory limit to 512Mi', got %s", cd.Status.RecommendedFix)
	}
	if cd.Status.Model != "gpt-4" {
		t.Errorf("Expected Model=gpt-4, got %s", cd.Status.Model)
	}
}
