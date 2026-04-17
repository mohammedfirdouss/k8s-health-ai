package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterDiagnosis stores LLM analysis for a failing Pod.
type ClusterDiagnosis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterDiagnosisSpec   `json:"spec,omitempty"`
	Status ClusterDiagnosisStatus `json:"status,omitempty"`
}

// ClusterDiagnosisSpec defines the diagnosed workload.
type ClusterDiagnosisSpec struct {
	TargetRef   TargetRef      `json:"targetRef"`
	FailureType string         `json:"failureType"`
	CollectedAt *metav1.Time   `json:"collectedAt,omitempty"`
}

// TargetRef points at the Pod that was analyzed.
type TargetRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ClusterDiagnosisStatus is written by the operator after LLM inference.
type ClusterDiagnosisStatus struct {
	Phase               string             `json:"phase,omitempty"`
	RootCause           string             `json:"rootCause,omitempty"`
	Severity            string             `json:"severity,omitempty"`
	RecommendedFix      string             `json:"recommendedFix,omitempty"`
	Model               string             `json:"model,omitempty"`
	LastUpdated         *metav1.Time       `json:"lastUpdated,omitempty"`
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
	Message             string             `json:"message,omitempty"`
	ObservedFingerprint string             `json:"observedFingerprint,omitempty"`
}

const (
	PhasePending   = "Pending"
	PhaseAnalyzing = "Analyzing"
	PhaseReady     = "Ready"
	PhaseError     = "Error"
)

// +kubebuilder:object:root=true

// ClusterDiagnosisList contains a list of ClusterDiagnosis.
type ClusterDiagnosisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterDiagnosis `json:"items"`
}
