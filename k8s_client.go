package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

//func main() {
	kubeconfig := getKubeConfigPath()
	fmt.Printf("Loading kubeconfig from: %s\n", kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating k8s client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespace := "kube-system" // Change this to the desired namespace
	fmt.Printf("\nFetching Pods in namespace '%s'.....\n\n", namespace)

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
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
