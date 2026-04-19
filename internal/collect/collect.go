package collect

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	healthv1alpha1 "github.com/k8s-health-ai/k8s-health-ai/api/v1alpha1"
	"github.com/k8s-health-ai/k8s-health-ai/internal/detect"
	"github.com/k8s-health-ai/k8s-health-ai/internal/llm"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const (
	maxEvents   = 20
	maxLogRunes = 12000
)

// Gather builds LLM context for a failing pod.
func Gather(ctx context.Context, k8s kubernetes.Interface, pod *corev1.Pod, container string, ft detect.FailureType) (llm.FailureContext, error) {
	specYAML, err := yaml.Marshal(pod.Spec)
	if err != nil {
		return llm.FailureContext{}, fmt.Errorf("marshal pod spec: %w", err)
	}

	events, err := k8s.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.kind=Pod", pod.Name, pod.Namespace),
	})
	if err != nil {
		return llm.FailureContext{}, fmt.Errorf("list events: %w", err)
	}
	evLines := summarizeEvents(events.Items, maxEvents)

	logTail, err := fetchLogs(ctx, k8s, pod, container, ft)
	if err != nil {
		logTail = "(logs unavailable: " + err.Error() + ")"
	}
	logTail = trimRunes(logTail, maxLogRunes)

	return llm.FailureContext{
		Namespace:   pod.Namespace,
		Pod:         pod.Name,
		Container:   container,
		FailureType: string(ft),
		PodSpecYAML: string(specYAML),
		Events:      evLines,
		LogsTail:    logTail,
	}, nil
}

func summarizeEvents(items []corev1.Event, limit int) []string {
	sort.Slice(items, func(i, j int) bool {
		ti := items[i].LastTimestamp.Time
		if ti.IsZero() {
			ti = items[i].EventTime.Time
		}
		tj := items[j].LastTimestamp.Time
		if tj.IsZero() {
			tj = items[j].EventTime.Time
		}
		return ti.After(tj)
	})
	out := make([]string, 0, len(items))
	for i, e := range items {
		if i >= limit {
			break
		}
		out = append(out, fmt.Sprintf("%s %s %s", e.Type, e.Reason, e.Message))
	}
	return out
}

func fetchLogs(ctx context.Context, k8s kubernetes.Interface, pod *corev1.Pod, container string, ft detect.FailureType) (string, error) {
	wantPrevious := ft == detect.CrashLoopBackOff || ft == detect.OOMKilled
	tail := int64(200)

	try := func(previous bool) (string, error) {
		opts := &corev1.PodLogOptions{
			Container: container,
			TailLines: &tail,
			Previous:  previous,
		}
		req := k8s.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, opts)
		stream, err := req.Stream(ctx)
		if err != nil {
			return "", err
		}
		defer stream.Close()
		var b strings.Builder
		_, err = io.Copy(&b, stream)
		return b.String(), err
	}

	if wantPrevious {
		s, err := try(true)
		if err == nil && strings.TrimSpace(s) != "" {
			return s, nil
		}
	}
	return try(false)
}

func trimRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "\n... (truncated)"
}

// BuildResourceUsage returns CPU/memory requests and limits for the named container
// (regular containers first, then init containers). Nil if none are set.
func BuildResourceUsage(pod *corev1.Pod, container string) *healthv1alpha1.ResourceUsage {
	if pod == nil {
		return nil
	}
	var c *corev1.Container
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == container {
			c = &pod.Spec.Containers[i]
			break
		}
	}
	if c == nil {
		for i := range pod.Spec.InitContainers {
			if pod.Spec.InitContainers[i].Name == container {
				c = &pod.Spec.InitContainers[i]
				break
			}
		}
	}
	if c == nil {
		return nil
	}
	ru := &healthv1alpha1.ResourceUsage{}
	if q, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
		ru.CpuRequest = q.String()
	}
	if q, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
		ru.CpuLimit = q.String()
	}
	if q, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
		ru.MemoryRequest = q.String()
	}
	if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
		ru.MemoryLimit = q.String()
	}
	if ru.CpuRequest == "" && ru.CpuLimit == "" && ru.MemoryRequest == "" && ru.MemoryLimit == "" {
		return nil
	}
	return ru
}
