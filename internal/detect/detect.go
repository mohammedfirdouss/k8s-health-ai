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
	CrashLoopBackOff FailureType = "CrashLoopBackOff"
	OOMKilled        FailureType = "OOMKilled"
	ImagePullBackOff FailureType = "ImagePullBackOff"
)

// Classify returns the primary failure for a pod, the container name, and ok if actionable.
func Classify(pod *corev1.Pod) (FailureType, string, bool) {
	if pod == nil {
		return "", "", false
	}
	// Prefer OOM over crash-loop when both appear (OOM is root cause).
	for _, cs := range pod.Status.ContainerStatuses {
		if term := cs.LastTerminationState.Terminated; term != nil && term.Reason == "OOMKilled" {
			return OOMKilled, cs.Name, true
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
	for _, cs := range pod.Status.ContainerStatuses {
		if w := cs.State.Waiting; w != nil && w.Reason == "CrashLoopBackOff" {
			return CrashLoopBackOff, cs.Name, true
		}
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
		return b.String()
	}
	return fmt.Sprintf("%s/%s|missing", pod.Namespace, pod.Name)
}

// DiagnosisName returns a deterministic metadata.name for ClusterDiagnosis.
func DiagnosisName(namespace, podName string, ft FailureType) string {
	h := sha256.Sum256([]byte(namespace + "/" + podName + "/" + string(ft)))
	return "diag-" + hex.EncodeToString(h[:])[:8]
}
