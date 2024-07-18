package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// initialize a kubernets client
	localConfig, err := clientcmd.BuildConfigFromFlags("", "/Users/tgoodwin/.kube/config")
	if err != nil {
		panic(err.Error())
	}
	c, err := kubernetes.NewForConfig(localConfig)
	pods, err := c.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Pod list resource version: ", pods.ResourceVersion)
	for _, pod := range pods.Items {
		uid := pod.GetUID()
		fmt.Println("Pod Name:", pod.Name, "UID:", uid, "RV:", pod.ResourceVersion)
	}
}
