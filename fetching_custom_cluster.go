package main

import (
	//"context"
	"context"
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
)

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
