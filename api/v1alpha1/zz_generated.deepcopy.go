package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies from in to out.
func (in *ClusterDiagnosis) DeepCopyInto(out *ClusterDiagnosis) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy returns a deep copy.
func (in *ClusterDiagnosis) DeepCopy() *ClusterDiagnosis {
	if in == nil {
		return nil
	}
	out := new(ClusterDiagnosis)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *ClusterDiagnosis) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ClusterDiagnosisSpec) DeepCopyInto(out *ClusterDiagnosisSpec) {
	*out = *in
	in.TargetRef.DeepCopyInto(&out.TargetRef)
	if in.CollectedAt != nil {
		in, out := &in.CollectedAt, &out.CollectedAt
		*out = new(metav1.Time)
		**out = **in
	}
}

func (in *TargetRef) DeepCopyInto(out *TargetRef) {
	*out = *in
}

func (in *ClusterDiagnosisStatus) DeepCopyInto(out *ClusterDiagnosisStatus) {
	*out = *in
	if in.LastUpdated != nil {
		in, out := &in.LastUpdated, &out.LastUpdated
		*out = new(metav1.Time)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto for ClusterDiagnosisList
func (in *ClusterDiagnosisList) DeepCopyInto(out *ClusterDiagnosisList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterDiagnosis, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *ClusterDiagnosisList) DeepCopy() *ClusterDiagnosisList {
	if in == nil {
		return nil
	}
	out := new(ClusterDiagnosisList)
	in.DeepCopyInto(out)
	return out
}

func (in *ClusterDiagnosisList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
