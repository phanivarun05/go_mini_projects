package deploymentmap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-mini-projects/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func InspectLineage(k8sClient *k8s.Client, namespace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pods, err := k8sClient.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing pods: %w", err)
	}

	helmMap := make(map[string][]string)
	argoMap := make(map[string][]string)
	var independentPods []string

	for _, pod := range pods.Items {
		var mapped bool
		podInfoStr := fmt.Sprintf("%s [Status: %s]", pod.Name, pod.Status.Phase)

		if trackingID, exists := pod.Annotations["argocd.argoproj.io/tracking-id"]; exists {
			parts := strings.Split(trackingID, ":")
			if len(parts) > 0 {
				argoMap[parts[0]] = append(argoMap[parts[0]], podInfoStr)
				mapped = true
			}
		}

		if !mapped {
			if argoInstance, exists := pod.Annotations["argocd.argoproj.io/instance"]; exists {
				argoMap[argoInstance] = append(argoMap[argoInstance], podInfoStr)
				mapped = true
			}
		}

		if !mapped {
			if instanceLabel, exists := pod.Labels["app.kubernetes.io/instance"]; exists {
				if helmRelease, isHelm := pod.Labels["meta.helm.sh/release-name"]; isHelm {
					helmMap[helmRelease] = append(helmMap[helmRelease], podInfoStr)
				} else {
					argoMap[instanceLabel] = append(argoMap[instanceLabel], podInfoStr)
				}
				mapped = true
			}
		}

		if !mapped {
			if helmRelease, exists := pod.Labels["meta.helm.sh/release-name"]; exists {
				helmMap[helmRelease] = append(helmMap[helmRelease], podInfoStr)
				mapped = true
			}
		}

		if !mapped {
			independentPods = append(independentPods, podInfoStr)
		}
	}

	fmt.Println("\n=========================================")
	fmt.Printf(" DEPLOYMENT TRACKING IN NAMESPACE: %s\n", namespace)
	fmt.Println("=========================================")

	if len(argoMap) > 0 {
		fmt.Println("\n🐙 ArgoCD Deployed Applications:")
		for appName, podList := range argoMap {
			fmt.Printf("   ├── 📱 App: %s (%d pods)\n", appName, len(podList))
			for _, p := range podList {
				fmt.Printf("   │   └── 🟢 Pod: %s\n", p)
			}
		}
	}

	if len(helmMap) > 0 {
		fmt.Println("\n📦 Native Helm Releases:")
		for release, podList := range helmMap {
			fmt.Printf("   ├── 📦 Release: %s (%d pods)\n", release, len(podList))
			for _, p := range podList {
				fmt.Printf("   │   └── 🟢 Pod: %s\n", p)
			}
		}
	}

	if len(independentPods) > 0 {
		fmt.Printf("\n⚙️  Independent / Unmanaged Pods (%d pods):\n", len(independentPods))
		for _, p := range independentPods {
			fmt.Printf("   └── 📄 Pod: %s\n", p)
		}
	}

	return nil
}
