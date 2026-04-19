package remediation

import "github.com/k8s-health-ai/k8s-health-ai/internal/detect"

// ForFailure returns short, actionable remediation hints for a failure class.
func ForFailure(ft detect.FailureType) []string {
	switch ft {
	case detect.OOMKilled:
		return []string{
			"Raise memory request/limit or remove the cap so the workload can use more RAM.",
			"Profile peak usage and set requests close to steady state, limits above peaks.",
			"Check for memory leaks or unbounded caches in the application.",
		}
	case detect.CrashLoopBackOff:
		return []string{
			"Inspect container logs and the previous instance log for the exit stack trace.",
			"Verify command, args, and env match what the image expects.",
			"Ensure readiness/liveness probes match real startup time.",
		}
	case detect.ImagePullBackOff:
		return []string{
			"Confirm image name, tag, and registry DNS from a node that can reach the registry.",
			"Check imagePullSecrets and registry credentials for private images.",
			"Validate platform/arch and that the tag exists (no typos).",
		}
	case detect.InitContainerError:
		return []string{
			"Read init container logs; fix the failing step before app containers run.",
			"Verify volumes, config maps, and secrets the init script needs are mounted.",
			"Increase init timeout or fix blocking network calls in the init.",
		}
	case detect.PendingScheduling:
		return []string{
			"Describe the pod for scheduler events (affinity, taints, resources, volumes).",
			"Relax node affinity or add capacity / new nodes if the cluster is full.",
			"Fix PVC binding or storage class if scheduling waits on volumes.",
		}
	default:
		return []string{
			"Describe the pod and recent events for the latest failure reason.",
			"Check container and init logs, then adjust resources, image, or config as indicated.",
		}
	}
}
