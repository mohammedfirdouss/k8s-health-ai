package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// FailureType is a supported pod failure class.
type FailureType string

const (
	CrashLoopBackOff   FailureType = "CrashLoopBackOff"
	OOMKilled          FailureType = "OOMKilled"
	ImagePullBackOff   FailureType = "ImagePullBackOff"
	InitContainerError FailureType = "InitContainerError"
	PendingScheduling  FailureType = "PendingScheduling"
)

func initContainerFailure(ics corev1.ContainerStatus) bool {
	if t := ics.State.Terminated; t != nil && t.ExitCode != 0 {
		return true
	}
	if w := ics.State.Waiting; w != nil {
		r := w.Reason
		if strings.Contains(r, "Error") {
			return true
		}
		switch r {
		case "CreateContainerConfigError", "ErrImagePull", "ImagePullBackOff",
			"InvalidImageName", "CrashLoopBackOff":
			return true
		}
	}
	return false
}

func classifyPendingScheduling(pod *corev1.Pod) (string, bool) {
	if pod.Status.Phase != corev1.PodPending {
		return "", false
	}
	var scheduled *corev1.PodCondition
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == corev1.PodScheduled {
			scheduled = &pod.Status.Conditions[i]
			break
		}
	}
	if scheduled != nil && scheduled.Status == corev1.ConditionFalse {
		switch scheduled.Reason {
		case "FailedScheduling", "Unschedulable":
			return "", true
		}
	}
	for _, ics := range pod.Status.InitContainerStatuses {
		if w := ics.State.Waiting; w != nil && w.Reason == "CreateContainerConfigError" {
			return ics.Name, true
		}
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if w := cs.State.Waiting; w != nil && w.Reason == "CreateContainerConfigError" {
			return cs.Name, true
		}
	}
	return "", false
}

// Classify returns the primary failure for a pod, the container name, and ok if actionable.
func Classify(pod *corev1.Pod) (FailureType, string, bool) {
	if pod == nil {
		return "", "", false
	}
	// Prefer OOM over everything when present (OOM is root cause).
	for _, cs := range pod.Status.ContainerStatuses {
		if term := cs.LastTerminationState.Terminated; term != nil && term.Reason == "OOMKilled" {
			return OOMKilled, cs.Name, true
		}
	}
	for _, ics := range pod.Status.InitContainerStatuses {
		if term := ics.LastTerminationState.Terminated; term != nil && term.Reason == "OOMKilled" {
			return OOMKilled, ics.Name, true
		}
	}
	for _, ics := range pod.Status.InitContainerStatuses {
		if initContainerFailure(ics) {
			return InitContainerError, ics.Name, true
		}
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if w := cs.State.Waiting; w != nil {
			switch w.Reason {
			case "ImagePullBackOff", "ErrImagePull":
				return ImagePullBackOff, cs.Name, true
			case "CrashLoopBackOff":
				return CrashLoopBackOff, cs.Name, true
			}
		}
	}
	if name, ok := classifyPendingScheduling(pod); ok {
		return PendingScheduling, name, true
	}
	return "", "", false
}

// Fingerprint hashes observable container state for the given container.
func Fingerprint(pod *corev1.Pod, container string) string {
	var b strings.Builder
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name != container {
			continue
		}
		writeContainerFingerprint(&b, cs)
		return b.String()
	}
	for _, ics := range pod.Status.InitContainerStatuses {
		if ics.Name != container {
			continue
		}
		writeContainerFingerprint(&b, ics)
		return b.String()
	}
	return fmt.Sprintf("%s/%s|missing", pod.Namespace, pod.Name)
}

func writeContainerFingerprint(b *strings.Builder, cs corev1.ContainerStatus) {
	b.WriteString(cs.Name)
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(int(cs.RestartCount)))
	if w := cs.State.Waiting; w != nil {
		b.WriteString("|W:")
		b.WriteString(w.Reason)
		b.WriteString(":")
		b.WriteString(w.Message)
	}
	if r := cs.LastTerminationState.Terminated; r != nil {
		b.WriteString("|T:")
		b.WriteString(r.Reason)
		b.WriteString(":")
		b.WriteString(strconv.FormatInt(int64(r.ExitCode), 10))
	}
}

// DiagnosisName returns a deterministic metadata.name for ClusterDiagnosis.
func DiagnosisName(namespace, podName string, ft FailureType) string {
	h := sha256.Sum256([]byte(namespace + "/" + podName + "/" + string(ft)))
	return "diag-" + hex.EncodeToString(h[:])[:8]
}
