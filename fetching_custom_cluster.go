package main

import (
	//"context"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	//"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func DebugLog(format string, v ...interface{}) {
	log.Printf("[HELM-DEBUG]"+format, v...)
}

func getKubeConfigPath() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Unable to find user home directory: %v", err)
	}
	return filepath.Join(home, ".kube", "config")
}

func ListReleases(actionConfig *action.Configuration, namespace, context string) {
	fmt.Println("--- Listing Helm Releases ---")

	listClient := action.NewList(actionConfig)
	listClient.AllNamespaces = false
	listClient.StateMask = action.ListDeployed | action.ListFailed

	releases, err := listClient.Run()
	if err != nil {
		log.Fatalf("Error listing Helm releases: %v", err)
	}

	if len(releases) == 0 {
		fmt.Printf("No Helm releases found in namespace '%s' of cluster '%s'.\n", namespace, context)
		return
	}

	fmt.Printf("%-20s | %-12s | %-10s | %-15s\n", "NAME", "STATUS", "REVISION", "CHART")
	fmt.Println("------------------------------------------------------------------")
	for _, release := range releases {
		fmt.Printf("%-20s | %-12s | %-10d | %-15s\n",
			release.Name,
			release.Info.Status.String(),
			release.Version,
			release.Chart.Name(),
		)
	}
}

func InstallOrUpgradeRelease(actionConfig *action.Configuration, releaseName, chartPath, namespace string, values map[string]interface{}) {
	fmt.Printf("\n--- Deploying Release '%s' ---\n", releaseName)

	chart, err := loader.Load(chartPath)
	if err != nil {
		log.Fatalf("Error loading chart from path '%s': %v", chartPath, err)
	}

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	_, err = histClient.Run(releaseName)

	if errors.Is(err, driver.ErrReleaseNotFound) {
		fmt.Printf("Release '%s' not found. Performing fresh INSTALL...\n", releaseName)

		installClient := action.NewInstall(actionConfig)
		installClient.ReleaseName = releaseName
		installClient.Namespace = namespace
		installClient.Timeout = 5 * time.Minute
		installClient.Wait = false

		rel, err := installClient.Run(chart, values)
		if err != nil {
			log.Fatalf("Error installing release '%s': %v", releaseName, err)
		}
		fmt.Printf("Successfully INSTALLED '%s' (Revision %d) | Status: %s\n",
			rel.Name, rel.Version, rel.Info.Status)
	} else if err == nil {
		fmt.Printf("Release '%s' found. Performing UPGRADE...\n", releaseName)

		upgradeClient := action.NewUpgrade(actionConfig)
		upgradeClient.Namespace = namespace
		upgradeClient.Timeout = 2 * time.Minute
		upgradeClient.Wait = false

		rel, err := upgradeClient.Run(releaseName, chart, values)
		if err != nil {
			log.Fatalf("Helm upgrade failed: %v", err)
		}

		fmt.Printf("Successfully UPGRADED '%s' to Revision %d | Status: %s\n",
			rel.Name, rel.Version, rel.Info.Status)
	} else {
		log.Fatalf("Error checking release history for '%s': %v", releaseName, err)
	}
}

func main() {
	kubeconfig := getKubeConfigPath()
	fmt.Printf("Loading Kubeconfig from %s\n", kubeconfig)

	rawConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		log.Fatalf("Error loading kubeconfig: %v", err)
	}

	fmt.Println("Available contexts (clusters):")
	for contextName := range rawConfig.Contexts {
		if contextName == rawConfig.CurrentContext {
			fmt.Printf("* %s (current)\n", contextName)
		} else {
			fmt.Printf("  %s\n", contextName)
		}
	}

	var targetContext, targetNamespace string

	fmt.Print("\nEnter the context (cluster) you want to use: ")
	fmt.Scanln(&targetContext)

	if _, exists := rawConfig.Contexts[targetContext]; !exists {
		log.Fatalf("Context '%s' not found in kubeconfig", targetContext)
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfig

	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: targetContext,
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		log.Fatalf("Error creating client config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	fmt.Printf("\nFetching available Namespaces from the Cluster: %s\n", targetContext)
	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error fetching namespaces: %v", err)
	}

	fmt.Printf("\nAvailable Namespaces:\n")
	for _, ns := range nsList.Items {
		fmt.Printf(" - %s\n", ns.Name)
	}

	fmt.Print("\nEnter the namespace you want to use (default is 'default'): ")
	fmt.Scanln(&targetNamespace)

	nsExists := false
	for _, ns := range nsList.Items {
		if ns.Name == targetNamespace {
			nsExists = true
			break
		}
	}

	if !nsExists {
		log.Fatalf("Namespace '%s' not found in the cluster", targetNamespace)
	}

	envSettings := cli.New()
	envSettings.KubeConfig = kubeconfig
	envSettings.KubeContext = targetContext

	actionConfig := new(action.Configuration)

	driver := "secret"

	if err := actionConfig.Init(envSettings.RESTClientGetter(), targetNamespace, driver, DebugLog); err != nil {
		log.Fatalf("Error initializing Helm action configuration: %v", err)
	}

	ListReleases(actionConfig, targetNamespace, targetContext)

	var chartPath, releaseName string
	fmt.Print("\nEnter the Helm chart path (e.g., ./mychart or stable/mysql): ")
	fmt.Scanln(&chartPath)

	fmt.Print("Enter the release name for the Helm chart: ")
	fmt.Scanln(&releaseName)

	customValues := map[string]interface{}{
		"replicaCount": 1, // Keep it light
		"image": map[string]interface{}{
			"repository": "nginx",
			"tag":        "alpine",
			"pullPolicy": "IfNotPresent",
		},
	}

	if _, err := os.Stat(chartPath); err == nil {
		InstallOrUpgradeRelease(actionConfig, releaseName, chartPath, targetNamespace, customValues)

		ListReleases(actionConfig, targetNamespace, targetContext)
	} else {
		log.Fatalf("Chart path '%s' does not exist", chartPath)
	}

	fmt.Printf("\nFetching pods from Namespace: %s in Cluster: %s\n", targetNamespace, targetContext)

	pods, err := clientset.CoreV1().Pods(targetNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error fetching pods: %v", err)
	}

	fmt.Printf("%-45s | %-12s | %-10s\n", "POD NAME", "PHASE", "RESTARTS")
	fmt.Println("-----------------------------------------------------------------------------")

	for _, pod := range pods.Items {
		totalRestarts := int32(0)
		for _, containerStatus := range pod.Status.ContainerStatuses {
			totalRestarts += containerStatus.RestartCount
		}

		fmt.Printf("%-45s | %-12s | %-10d\n",
			pod.Name,
			pod.Status.Phase,
			totalRestarts)
	}

}
