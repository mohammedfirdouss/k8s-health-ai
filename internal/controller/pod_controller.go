package controller

import (
	"context"
	"time"

	healthv1alpha1 "github.com/k8s-health-ai/k8s-health-ai/api/v1alpha1"
	"github.com/k8s-health-ai/k8s-health-ai/internal/collect"
	"github.com/k8s-health-ai/k8s-health-ai/internal/detect"
	"github.com/k8s-health-ai/k8s-health-ai/internal/llm"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	Client client.Client
	K8s    kubernetes.Interface
	Scheme *runtime.Scheme
	LLM    llm.Provider
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
	var cd healthv1alpha1.ClusterDiagnosis
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: name}, &cd)
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
			return ctrl.Result{}, err
		}
		if err := r.Client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: name}, &cd); err != nil {
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
	cd.Status.Phase = healthv1alpha1.PhaseAnalyzing
	cd.Status.Message = ""
	if err := r.Client.Status().Update(ctx, &cd); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: name}, &cd); err != nil {
		return ctrl.Result{}, err
	}

	fc, err := collect.Gather(ctx, r.K8s, &pod, container, ft)
	if err != nil {
		return r.fail(ctx, &cd, err)
	}
	fc.CollectedAt = now

	diag, err := r.LLM.Diagnose(ctx, fc)
	if err != nil {
		return r.fail(ctx, &cd, err)
	}

	cd.Status.Phase = healthv1alpha1.PhaseReady
	cd.Status.RootCause = diag.RootCause
	cd.Status.Severity = diag.Severity
	cd.Status.RecommendedFix = diag.RecommendedFix
	cd.Status.Model = diag.Model
	cd.Status.LastUpdated = &now
	cd.Status.ObservedFingerprint = fp
	cd.Status.Message = ""

	if cd.Annotations == nil {
		cd.Annotations = map[string]string{}
	}
	cd.Annotations[lastLLMCallAnnotation] = time.Now().UTC().Format(time.RFC3339)
	if err := r.Client.Update(ctx, &cd); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Status().Update(ctx, &cd); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PodReconciler) fail(ctx context.Context, cd *healthv1alpha1.ClusterDiagnosis, err error) (ctrl.Result, error) {
	cd.Status.Phase = healthv1alpha1.PhaseError
	cd.Status.Message = err.Error()
	_ = r.Client.Status().Update(ctx, cd)
	return ctrl.Result{}, err
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
