// Command diagctl lists, inspects, and deletes ClusterDiagnosis resources (non-interactive).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	healthv1alpha1 "github.com/k8s-health-ai/k8s-health-ai/api/v1alpha1"
	"github.com/k8s-health-ai/k8s-health-ai/internal/detect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(healthv1alpha1.AddToScheme(scheme))
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printGlobalUsage(os.Stderr)
		os.Exit(1)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printGlobalUsage(os.Stdout)
		return
	}

	cmd := args[0]
	rest := args[1:]
	var err error
	switch cmd {
	case "list":
		err = runList(rest)
	case "get":
		err = runGet(rest)
	case "delete":
		err = runDelete(rest)
	case "explain":
		err = runExplain(rest)
	default:
		fmt.Fprintf(os.Stderr, "diagctl: unknown command %q\n\n", cmd)
		printGlobalUsage(os.Stderr)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "diagctl: %v\n", err)
		os.Exit(1)
	}
}

type globalFlags struct {
	namespace string
	output    string
}

func parseGlobal(fs *flag.FlagSet, g *globalFlags) {
	fs.StringVar(&g.namespace, "n", "", "namespace (empty = all namespaces for list; get/delete resolve by name when omitted)")
	fs.StringVar(&g.output, "o", "text", "output format: text or json")
}

func validateOutput(o string) error {
	switch o {
	case "text", "json":
		return nil
	default:
		return fmt.Errorf("invalid -o %q (use text or json)", o)
	}
}

func kubeClient(ctx context.Context) (client.Client, error) {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("kubernetes config (in-cluster or kubeconfig): %w", err)
	}
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("controller-runtime client: %w", err)
	}
	return c, nil
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var g globalFlags
	parseGlobal(fs, &g)
	fs.Usage = func() { printListUsage(os.Stderr) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := validateOutput(g.output); err != nil {
		return err
	}
	for _, a := range fs.Args() {
		if a == "-h" || a == "--help" {
			printListUsage(os.Stdout)
			return nil
		}
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %v\n\n", fs.Args())
		printListUsage(os.Stderr)
		return fmt.Errorf("invalid usage")
	}

	ctx := context.Background()
	c, err := kubeClient(ctx)
	if err != nil {
		return err
	}

	var list healthv1alpha1.ClusterDiagnosisList
	opts := []client.ListOption{}
	if g.namespace != "" {
		opts = append(opts, client.InNamespace(g.namespace))
	}
	if err := c.List(ctx, &list, opts...); err != nil {
		return fmt.Errorf("list clusterdiagnoses: %w", err)
	}

	switch g.output {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(&list)
	default:
		return printListText(os.Stdout, list.Items)
	}
}

func printListText(w io.Writer, items []healthv1alpha1.ClusterDiagnosis) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAMESPACE\tNAME\tPHASE\tFAILURE\tPOD\tAGE")
	for _, item := range items {
		ns := item.Namespace
		phase := item.Status.Phase
		ft := item.Spec.FailureType
		pod := item.Spec.TargetRef.Name
		age := ageString(item.CreationTimestamp.Time)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", ns, item.Name, phase, ft, pod, age)
	}
	_ = tw.Flush()
}

func ageString(t time.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	s := d.Round(time.Second).String()
	if strings.HasSuffix(s, "0s") && d >= time.Minute {
		s = strings.TrimSuffix(s, "0s")
	}
	return s + " ago"
}

func runGet(args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var g globalFlags
	parseGlobal(fs, &g)
	fs.Usage = func() { printGetUsage(os.Stderr) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := validateOutput(g.output); err != nil {
		return err
	}
	for _, a := range fs.Args() {
		if a == "-h" || a == "--help" {
			printGetUsage(os.Stdout)
			return nil
		}
	}
	name := fs.Arg(0)
	if name == "" {
		printGetUsage(os.Stderr)
		return fmt.Errorf("NAME is required")
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %v\n\n", fs.Args()[1:])
		printGetUsage(os.Stderr)
		return fmt.Errorf("invalid usage")
	}

	ctx := context.Background()
	c, err := kubeClient(ctx)
	if err != nil {
		return err
	}

	cd, err := resolveDiagnosis(ctx, c, g.namespace, name)
	if err != nil {
		return err
	}

	switch g.output {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cd)
	default:
		out, err := yaml.Marshal(cd)
		if err != nil {
			return fmt.Errorf("marshal yaml: %w", err)
		}
		_, err = os.Stdout.Write(out)
		return err
	}
}

func resolveDiagnosis(ctx context.Context, c client.Client, namespace, name string) (*healthv1alpha1.ClusterDiagnosis, error) {
	if namespace != "" {
		var cd healthv1alpha1.ClusterDiagnosis
		err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &cd)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("clusterdiagnosis %q not found in namespace %q", name, namespace)
			}
			return nil, err
		}
		return &cd, nil
	}

	var list healthv1alpha1.ClusterDiagnosisList
	if err := c.List(ctx, &list); err != nil {
		return nil, err
	}
	var matches []healthv1alpha1.ClusterDiagnosis
	for i := range list.Items {
		if list.Items[i].Name == name {
			matches = append(matches, list.Items[i])
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("clusterdiagnosis %q not found in any namespace (try: diagctl get %s -n <namespace>)", name, name)
	case 1:
		return &matches[0], nil
	default:
		var nss []string
		for _, m := range matches {
			nss = append(nss, m.Namespace)
		}
		return nil, fmt.Errorf("multiple clusterdiagnoses named %q in namespaces %v; specify -n", name, nss)
	}
}

func runDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var g globalFlags
	var yes bool
	parseGlobal(fs, &g)
	fs.BoolVar(&yes, "yes", false, "required to perform delete (non-interactive safety)")
	fs.Usage = func() { printDeleteUsage(os.Stderr) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := validateOutput(g.output); err != nil {
		return err
	}
	for _, a := range fs.Args() {
		if a == "-h" || a == "--help" {
			printDeleteUsage(os.Stdout)
			return nil
		}
	}
	name := fs.Arg(0)
	if name == "" {
		printDeleteUsage(os.Stderr)
		return fmt.Errorf("NAME is required")
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %v\n\n", fs.Args()[1:])
		printDeleteUsage(os.Stderr)
		return fmt.Errorf("invalid usage")
	}

	if !yes {
		return fmt.Errorf("refusing to delete without --yes; example: diagctl delete NAME -n NS --yes")
	}

	ctx := context.Background()
	c, err := kubeClient(ctx)
	if err != nil {
		return err
	}

	cd, err := resolveDiagnosis(ctx, c, g.namespace, name)
	if err != nil {
		return err
	}

	if err := c.Delete(ctx, cd); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("clusterdiagnosis %q not found", name)
		}
		return fmt.Errorf("delete: %w", err)
	}
	fmt.Fprintf(os.Stdout, "clusterdiagnosis %q deleted from namespace %q\n", cd.Name, cd.Namespace)
	return nil
}

func runExplain(args []string) error {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { printExplainUsage(os.Stderr) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(fs.Args()) > 0 {
		if fs.Args()[0] == "-h" || fs.Args()[0] == "--help" {
			printExplainUsage(os.Stdout)
			return nil
		}
		fmt.Fprintf(os.Stderr, "unexpected arguments: %v\n\n", fs.Args())
		printExplainUsage(os.Stderr)
		return fmt.Errorf("invalid usage")
	}
	printExplainText(os.Stdout)
	return nil
}

func printExplainText(w io.Writer) {
	var b strings.Builder
	b.WriteString("LLM environment variables (operator / manager)\n\n")
	b.WriteString("  LLM_PROVIDER\n")
	b.WriteString("    Selects the LLM backend: mock (default), bedrock, vertex, openai, azure-openai, ollama.\n\n")
	b.WriteString("  LLM_RPM\n")
	b.WriteString("    Optional requests-per-minute cap for LLM calls (default 120). Set to 0 or negative to disable limiting.\n\n")
	b.WriteString("Provider-specific configuration is read by the manager (not diagctl), for example:\n")
	b.WriteString("  AWS_REGION, BEDROCK_MODEL_ID (Bedrock); VERTEX_MODEL (Vertex);\n")
	b.WriteString("  OPENAI_API_KEY, OPENAI_BASE_URL, OPENAI_MODEL (OpenAI);\n")
	b.WriteString("  AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_API_KEY, AZURE_OPENAI_DEPLOYMENT (Azure OpenAI);\n")
	b.WriteString("  OLLAMA_HOST, OLLAMA_MODEL (Ollama).\n\n")
	b.WriteString("Failure types (spec.failureType on ClusterDiagnosis)\n\n")
	b.WriteString("  These are the primary failure classes the operator detects from Pods:\n\n")
	b.WriteString(fmt.Sprintf("  %s — container exits and restarts repeatedly.\n", detect.CrashLoopBackOff))
	b.WriteString(fmt.Sprintf("  %s — process terminated due to memory limit.\n", detect.OOMKilled))
	b.WriteString(fmt.Sprintf("  %s — image cannot be pulled (includes ErrImagePull / registry issues).\n", detect.ImagePullBackOff))
	b.WriteString(fmt.Sprintf("  %s — init container failed or stuck in a bad waiting state.\n", detect.InitContainerError))
	b.WriteString(fmt.Sprintf("  %s — pod pending with scheduling or config issues (e.g. unschedulable).\n", detect.PendingScheduling))
	_, _ = io.WriteString(w, b.String())
}

func printGlobalUsage(w io.Writer) {
	var b strings.Builder
	b.WriteString("diagctl — list, get, delete, or explain ClusterDiagnosis resources (health.k8sai.io/v1alpha1).\n\n")
	b.WriteString("Usage:\n")
	b.WriteString("  diagctl <command> [arguments]\n\n")
	b.WriteString("Commands:\n")
	b.WriteString("  list      List ClusterDiagnosis objects (kubectl-style columns or JSON)\n")
	b.WriteString("  get       Print one ClusterDiagnosis as YAML (text) or JSON\n")
	b.WriteString("  delete    Delete a ClusterDiagnosis (requires --yes)\n")
	b.WriteString("  explain   Static help for LLM_* env vars and failure types (no cluster)\n\n")
	b.WriteString("Global flags (list, get, delete):\n")
	b.WriteString("  -n string    Namespace; empty lists all namespaces. get/delete resolve by name across namespaces if omitted.\n")
	b.WriteString("  -o string     Output: text (default) or json\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("  diagctl list\n")
	b.WriteString("  diagctl list -n kube-system -o json\n")
	b.WriteString("  diagctl get my-diagnosis-abc -n kube-system\n")
	b.WriteString("  diagctl get my-diagnosis-abc\n")
	b.WriteString("  diagctl delete my-diagnosis-abc -n kube-system --yes\n")
	b.WriteString("  diagctl explain\n")
	_, _ = io.WriteString(w, b.String())
}

func printListUsage(w io.Writer) {
	var b strings.Builder
	b.WriteString("Usage: diagctl list [-n namespace] [-o text|json]\n\n")
	b.WriteString("List ClusterDiagnosis resources. Uses the current kubeconfig or in-cluster credentials.\n\n")
	b.WriteString("Flags:\n")
	b.WriteString("  -n string   Namespace filter; omit to include all namespaces\n")
	b.WriteString("  -o string   text (table) or json (ClusterDiagnosisList); default text\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("  diagctl list\n")
	b.WriteString("  diagctl list -n kube-system\n")
	b.WriteString("  diagctl list -o json\n")
	_, _ = io.WriteString(w, b.String())
}

func printGetUsage(w io.Writer) {
	var b strings.Builder
	b.WriteString("Usage: diagctl get [-n namespace] [-o text|json] NAME\n\n")
	b.WriteString("Print one ClusterDiagnosis. With -o text, output is YAML. With -o json, output is JSON.\n\n")
	b.WriteString("Flags:\n")
	b.WriteString("  -n string   Namespace; if omitted, NAME must be unique cluster-wide\n")
	b.WriteString("  -o string   text (YAML) or json; default text\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("  diagctl get my-diagnosis-abc -n kube-system\n")
	b.WriteString("  diagctl get my-diagnosis-abc -o json -n kube-system\n")
	_, _ = io.WriteString(w, b.String())
}

func printDeleteUsage(w io.Writer) {
	var b strings.Builder
	b.WriteString("Usage: diagctl delete [-n namespace] [-o text|json] --yes NAME\n\n")
	b.WriteString("Delete a ClusterDiagnosis. Destructive; requires --yes (non-interactive).\n\n")
	b.WriteString("Flags:\n")
	b.WriteString("  -n string    Namespace; if omitted, NAME must be unique cluster-wide\n")
	b.WriteString("  -o string     Ignored for delete (reserved for consistency)\n")
	b.WriteString("  --yes        Required to actually delete\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("  diagctl delete my-diagnosis-abc -n kube-system --yes\n")
	_, _ = io.WriteString(w, b.String())
}

func printExplainUsage(w io.Writer) {
	var b strings.Builder
	b.WriteString("Usage: diagctl explain\n\n")
	b.WriteString("Print static reference for LLM_* environment variables and failure types. No Kubernetes access.\n\n")
	b.WriteString("Examples:\n")
	b.WriteString("  diagctl explain\n")
	_, _ = io.WriteString(w, b.String())
}
