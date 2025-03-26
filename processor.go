package main

import (
	"context"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Processor struct{}

type ServiceSpec struct {
	CPU        int
	GPU_slices map[string]int
	Memory     int
	Env        map[string]string
	Name       string
	Model      string
	Label      map[string]string
}

type ResourceEstimate struct {
	CPU        int
	GPU_slices map[string]int
	Memory     int
}

// DecideService fills the remaining information in the ServiceSpec based on the RequestGroup and ResourceEstimate
func (d Processor) DecideService(group RequestGroup) ServiceSpec {
	log.Println("Deciding service spec based on request group")
	resourceEstimate := d.ResourceEstimate(group, gpuMode)

	spec := ServiceSpec{
		CPU:        resourceEstimate.CPU,
		GPU_slices: resourceEstimate.GPU_slices,
		Memory:     resourceEstimate.Memory,
		Env:        group.Requests[0].Env,
		Name:       group.Requests[0].Model,
		Model:      group.Requests[0].Model,
		Label:      group.Requests[0].Label,
	}
	//log.Printf("Decided ServiceSpec - CPU: %d, GPU: %d, Memory: %d, ServiceName: %s, Model: %s, SLO: %d", spec.CPU, spec.GPU, spec.Memory, spec.ServiceName, spec.Model, spec.SLO)
	return spec
}

// Estimate Resource usage for a RequestGroup
func (d Processor) ResourceEstimate(group RequestGroup, gpuMode string) ResourceEstimate {
	// Policy , gives smallest slice availablem on cluster.
	log.Println("Estimating resources for request group")

	var ConfigList []string
	var ConfigMap map[string]int

	if gpuMode == "mps" {
		log.Println("Current GPU mode is MPS")

		ConfigList = []string{}
		for i := 1; i <= 32; i++ {
			ConfigList = append(ConfigList, fmt.Sprintf("nvidia.com/gpu-%dgb", i))
		}
		ConfigMap = make(map[string]int)
		for _, config := range ConfigList {
			ConfigMap[config] = 0
		}

	} else if gpuMode == "mig" {
		log.Println("Current GPU mode is MIG")
		ConfigMap = map[string]int{
			"nvidia.com/mig-1g.5gb":  0,
			"nvidia.com/mig-2g.10gb": 0,
			"nvidia.com/mig-3g.20gb": 0,
			"nvidia.com/mig-4g.20gb": 0,
			"nvidia.com/mig-7g.40gb": 0,
			"nvidia.com/gpu":         0,
		}

		ConfigList = []string{
			"nvidia.com/mig-1g.5gb",
			"nvidia.com/mig-2g.10gb",
			"nvidia.com/mig-3g.20gb",
			"nvidia.com/mig-4g.20gb",
			"nvidia.com/mig-7g.40gb",
			"nvidia.com/gpu",
		}
	}

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
		for _, migConfig := range ConfigList {
			if Quantity, ok := node.Status.Capacity[v1.ResourceName(migConfig)]; ok {
				ConfigMap[migConfig] += int(Quantity.Value())
			}
		}
	}
	log.Printf("Available %s slices: %v", gpuMode, ConfigMap)

	// Find the smallest available GPU slice
	smallestSlice := ""

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
