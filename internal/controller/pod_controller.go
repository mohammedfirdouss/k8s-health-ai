package controller

import (
	"context"
	"fmt"
	"time"

	healthv1alpha1 "github.com/k8s-health-ai/k8s-health-ai/api/v1alpha1"
	"github.com/k8s-health-ai/k8s-health-ai/internal/collect"
	"github.com/k8s-health-ai/k8s-health-ai/internal/detect"
	"github.com/k8s-health-ai/k8s-health-ai/internal/llm"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	lastLLMCallAnnotation = "health.k8sai.io/last-llm-call"
	minBetweenLLM         = 5 * time.Minute
)

// PodReconciler creates ClusterDiagnosis for failing pods.
type PodReconciler struct {
	Client    client.Client
	APIReader client.Reader
	K8s       kubernetes.Interface
	Scheme    *runtime.Scheme
	LLM       llm.Provider
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch
// +kubebuilder:rbac:groups=health.k8sai.io,resources=clusterdiagnoses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=health.k8sai.io,resources=clusterdiagnoses/status,verbs=get;update;patch

// Reconcile implements ctrl.Reconciler.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pod corev1.Pod
	if err := r.Client.Get(ctx, req.NamespacedName, &pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ft, container, ok := detect.Classify(&pod)
	if !ok {
		return ctrl.Result{}, nil
	}

	name := detect.DiagnosisName(pod.Namespace, pod.Name, ft)
	key := client.ObjectKey{Namespace: pod.Namespace, Name: name}
	var cd healthv1alpha1.ClusterDiagnosis
	err := r.Client.Get(ctx, key, &cd)
	if apierrors.IsNotFound(err) {
		cd = healthv1alpha1.ClusterDiagnosis{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: pod.Namespace,
			},
			Spec: healthv1alpha1.ClusterDiagnosisSpec{
				TargetRef: healthv1alpha1.TargetRef{
					Kind:      "Pod",
					Namespace: pod.Namespace,
					Name:      pod.Name,
				},
				FailureType: string(ft),
			},
		}
		if err := controllerutil.SetControllerReference(&pod, &cd, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Client.Create(ctx, &cd); err != nil {
			if apierrors.IsAlreadyExists(err) {
				if err := r.getDiagnosis(ctx, key, &cd); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				return ctrl.Result{}, err
			}
		}
		// Uncached read: the delegating cache may not see a just-created object yet.
		if err := r.getDiagnosis(ctx, key, &cd); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	fp := detect.Fingerprint(&pod, container)
	if cd.Status.Phase == healthv1alpha1.PhaseReady && cd.Status.ObservedFingerprint == fp {
		return ctrl.Result{}, nil
	}

	if t := cd.Annotations[lastLLMCallAnnotation]; t != "" {
		if ts, e := time.Parse(time.RFC3339, t); e == nil {
			if elapsed := time.Since(ts); elapsed < minBetweenLLM {
				return ctrl.Result{RequeueAfter: minBetweenLLM - elapsed}, nil
			}
		}
	}

	now := metav1.Now()
	if err := r.patchDiagnosisStatus(ctx, key, func(cur *healthv1alpha1.ClusterDiagnosis) {
		cur.Status.Phase = healthv1alpha1.PhaseAnalyzing
		cur.Status.Message = ""
	}); err != nil {
		return ctrl.Result{}, err
	}

	fc, err := collect.Gather(ctx, r.K8s, &pod, container, ft)
	if err != nil {
		return r.fail(ctx, req.NamespacedName, name, err)
	}

	diag, err := r.LLM.Diagnose(ctx, fc)
	if err != nil {
		return r.fail(ctx, req.NamespacedName, name, err)
	}

	if err := r.patchDiagnosisStatus(ctx, key, func(cur *healthv1alpha1.ClusterDiagnosis) {
		cur.Status.Phase = healthv1alpha1.PhaseReady
		cur.Status.RootCause = diag.RootCause
		cur.Status.Severity = diag.Severity
		cur.Status.RecommendedFix = diag.RecommendedFix
		cur.Status.Model = diag.Model
		cur.Status.LastUpdated = &now
		cur.Status.ObservedFingerprint = fp
		cur.Status.Message = ""
	}); err != nil {
		return ctrl.Result{}, err
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	if err := r.patchDiagnosisMeta(ctx, key, func(cur *healthv1alpha1.ClusterDiagnosis) {
		if cur.Annotations == nil {
			cur.Annotations = map[string]string{}
		}
		cur.Annotations[lastLLMCallAnnotation] = ts
	}); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PodReconciler) fail(ctx context.Context, podNN types.NamespacedName, cdName string, reconcileErr error) (ctrl.Result, error) {
	key := client.ObjectKey{Namespace: podNN.Namespace, Name: cdName}
	msg := reconcileErr.Error()
	_ = r.patchDiagnosisStatus(ctx, key, func(cur *healthv1alpha1.ClusterDiagnosis) {
		cur.Status.Phase = healthv1alpha1.PhaseError
		cur.Status.Message = msg
	})
	return ctrl.Result{}, reconcileErr
}

func (r *PodReconciler) getDiagnosis(ctx context.Context, key client.ObjectKey, cd *healthv1alpha1.ClusterDiagnosis) error {
	if r.APIReader != nil {
		return r.APIReader.Get(ctx, key, cd)
	}
	return r.Client.Get(ctx, key, cd)
}

func (r *PodReconciler) patchDiagnosisStatus(ctx context.Context, key client.ObjectKey, mutate func(*healthv1alpha1.ClusterDiagnosis)) error {
	var last error
	for attempt := 0; attempt < 8; attempt++ {
		var cur healthv1alpha1.ClusterDiagnosis
		if err := r.getDiagnosis(ctx, key, &cur); err != nil {
			return err
		}
		mutate(&cur)
		last = r.Client.Status().Update(ctx, &cur)
		if last == nil {
			return nil
		}
		if apierrors.IsConflict(last) || apierrors.IsTimeout(last) {
			time.Sleep(time.Duration(25*(attempt+1)) * time.Millisecond)
			continue
		}
		return last
	}
	return fmt.Errorf("status update retries exhausted: %w", last)
}

func (r *PodReconciler) patchDiagnosisMeta(ctx context.Context, key client.ObjectKey, mutate func(*healthv1alpha1.ClusterDiagnosis)) error {
	var last error
	for attempt := 0; attempt < 8; attempt++ {
		var cur healthv1alpha1.ClusterDiagnosis
		if err := r.getDiagnosis(ctx, key, &cur); err != nil {
			return err
		}
		mutate(&cur)
		last = r.Client.Update(ctx, &cur)
		if last == nil {
			return nil
		}
		if apierrors.IsConflict(last) || apierrors.IsTimeout(last) {
			time.Sleep(time.Duration(25*(attempt+1)) * time.Millisecond)
			continue
		}
		return last
	}
	return fmt.Errorf("metadata update retries exhausted: %w", last)
}

// SetupWithManager registers the reconciler.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("pod-health").
		For(&corev1.Pod{}).
		WithEventFilter(failurePredicate()).
		Complete(r)
}

func failurePredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return false
		}
		_, _, ok = detect.Classify(pod)
		return ok
	})
}
