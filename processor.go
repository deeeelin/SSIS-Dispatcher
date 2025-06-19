package main

import (
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Processor struct{}

type ServiceSpec struct {
	CPU              int
	GPU_slices       map[string]int
	Memory           int
	Env              map[string]string
	Name             string
	Model            string
	Label            map[string]string
	RuntimeClassName string
}

type ResourceEstimate struct {
	CPU        int
	GPU_slices map[string]int
	Memory     int
}

// DecideService fills the remaining information in the ServiceSpec based on the RequestGroup and ResourceEstimate
func (d Processor) DecideService(group RequestGroup) ServiceSpec {
	log.Println("Deciding service spec based on request group")
	resourceEstimate := d.ResourceEstimate(group)

	spec := ServiceSpec{
		CPU:              resourceEstimate.CPU,
		GPU_slices:       resourceEstimate.GPU_slices,
		Memory:           resourceEstimate.Memory,
		Env:              group.Requests[0].Env,
		Name:             group.Requests[0].Model,
		Model:            group.Requests[0].Model,
		Label:            group.Requests[0].Label,
		RuntimeClassName: "nvidia",
	}
	//log.Printf("Decided ServiceSpec - CPU: %d, GPU: %d, Memory: %d, ServiceName: %s, Model: %s, SLO: %d", spec.CPU, spec.GPU, spec.Memory, spec.ServiceName, spec.Model, spec.SLO)
	return spec
}

// Estimate Resource usage for a RequestGroup
func (d Processor) ResourceEstimate(group RequestGroup) ResourceEstimate {
	// Policy , gives smallest slice availablem on cluster.
	log.Println("Estimating resources for request group")

	var totalCPU int
	var totalMemory int

	//CPU, Memory logic define here
	totalCPU = 4000      // 10 CPUs, TGI requires massive ammount of cpu and memory , or else there will be error occured
	totalMemory = 102400 // 100GB , TGI requires massive ammount of cpu and memory , or else there will be error occured

	// GPU logic define here //
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list nodes: %v", err)
	}

	for _, node := range nodes.Items {
		for _, gpuConfig := range ConfigList {
			if Quantity, ok := node.Status.Capacity[v1.ResourceName(gpuConfig)]; ok {
				ConfigMap[gpuConfig] += int(Quantity.Value())
			}
		}
	}
	log.Printf("Available GPU resources: %v", ConfigMap)

	// Find the smallest available GPU slice
	smallestSlice := ConfigList[0] // Default to the first config in case none are available

	for _, config := range ConfigList {
		if ConfigMap[config] > 0 {
			smallestSlice = config
			break
		}
	}
	log.Print("Assigned resources , CPU : ", totalCPU, " Memory : ", totalMemory, " GPU : ", smallestSlice)

	return ResourceEstimate{
		CPU:        totalCPU,                         // Total CPU estimate
		GPU_slices: map[string]int{smallestSlice: 1}, // Set to 0 for now, unless GPU resources are also required
		Memory:     totalMemory,                      // Total Memory estimate
	}
}
