package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/client/pkg/kn/commands"
	servinglib "knative.dev/client/pkg/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

type Assigner struct{}

// create the service and forward the request
func (a *Assigner) AssignService(spec ServiceSpec, group RequestGroup) {
	log.Println("Assigning service based on the ServiceSpec")
	p := commands.KnParams{}
	p.Initialize()

	// Process each request in group to json payloads , store in array
	var packedRequests []io.ReadCloser
	for _, req := range group.Requests {
		packedRequest := a.CreatePayload(req)
		packedRequests = append(packedRequests, packedRequest)
		log.Printf("Packed request for token: %s", req.Token)
	}

	// Initialize the Knative serving client
	client, err := p.NewServingClient(namespace)
	if err != nil {
		log.Fatalf("Error creating Knative serving client: %s", err.Error())
	}

	// List all services
	serviceList, err := client.ListServices(context.Background())
	if err != nil {
		log.Fatalf("Error listing Knative services: %s", err.Error())
	}

	// Check if the specified service name from spec exists
	serviceExists := false
	for _, svc := range serviceList.Items {
		log.Printf("Found service: %s", svc.Name)
		if svc.Name == spec.Name {
			serviceExists = true
			break
		}
	}

	// There is a service running , just forward the payload
	if serviceExists {
		log.Printf("Service %s exists, updating the service", spec.Name)
		a.CurrentService(spec, packedRequests)
	} else {
		// There isn't service running ,create a service and forward payload
		log.Printf("Service %s does not exist, creating a new service", spec.Name)
		a.CreateNewService(spec, packedRequests)
	}
}

// CreatePayload creates a single json payload for a request
func (a *Assigner) CreatePayload(req Request) io.ReadCloser {
	log.Println("Creating payload for request")

	payload := map[string]interface{}{
		"inputs":     req.Token,
		"parameters": req.Par,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Error packing request: %s", err.Error())
	}
	log.Printf("Request Payload: %s", string(jsonPayload))
	return ioutil.NopCloser(bytes.NewBuffer(jsonPayload))
}

// CreateNewService creates a new service and forwards the request
func (a *Assigner) CreateNewService(spec ServiceSpec, requestPayloads []io.ReadCloser) {
	log.Printf("Creating new Knative service - Name: %s", spec.Name)
	p := commands.KnParams{}
	p.Initialize()

	// Create new knative serving client
	client, err := p.NewServingClient(namespace)
	if err != nil {
		log.Fatalf("Error creating Knative serving client: %s", err.Error())
	}

	//Create a service instance
	var svcInstance = &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: namespace,
		},
	}

	// Define resource requirements based on the spec
	resourceRequirements := v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", spec.CPU)),     // Convert to millicores
			v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", spec.Memory)), // Memory in MiB
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", spec.CPU)),
			v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", spec.Memory)),
		},
	}

	// Add GPU resource request if specified on spec
	if len(spec.GPU_slices) > 0 {
		for sliceType, quantity := range spec.GPU_slices {
			resourceRequirements.Requests[v1.ResourceName(sliceType)] = resource.MustParse(fmt.Sprintf("%d", quantity))
			resourceRequirements.Limits[v1.ResourceName(sliceType)] = resource.MustParse(fmt.Sprintf("%d", quantity))
		}
	}

	// Select the image and command based on the spec.Model on Image Map
	var selectedImage map[string]string
	if spec.Model == "test" {
		selectedImage = testImage
	} else {
		selectedImage = inferImage
	}
	image := selectedImage["image"]
	command := selectedImage["command"]
	fmt.Println("Choosed image", image)

	// Convert spec.Env from a map[string]string to []v1.EnvVar
	var envVars []v1.EnvVar
	for key, value := range spec.Env {
		envVars = append(envVars, v1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	log.Printf("Environment variables for service: %+v", envVars)

	// Add all the resource requirements define to service instance
	svcInstance.Spec.Template = servingv1.RevisionTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				servinglib.UserImageAnnotationKey: "",
				//	"autoscaling.knative.dev/max-scale": "1", // BUG HERE , not working // default let autoscaler at most scale to one
			},
			Labels: spec.Label,
		},
		Spec: servingv1.RevisionSpec{
			PodSpec: v1.PodSpec{
				Containers: []v1.Container{{
					Image:           image,
					ImagePullPolicy: v1.PullAlways,
					Resources:       resourceRequirements,
					// VolumeMounts: []v1.VolumeMount{{
					// 	Name:      "disk-volume",
					// 	MountPath: "/data",
					// }},
					Env: envVars,
				}},
				// Volumes: []v1.Volume{{
				// 	Name: "disk-volume",
				// 	VolumeSource: v1.VolumeSource{
				// 		PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				// 			ClaimName: "knative-pv-claim", // Use the PVC created in your cluster
				// 			ReadOnly:  false,              // Set to false if you need write access
				// 		},
				// 	},
				// }},
			},
		},
	}

	if command != "" {
		log.Printf("Setting command: %s", command)
		svcInstance.Spec.Template.Spec.PodSpec.Containers[0].Command = []string{command}
	}

	// Use the service instance to create service
	ctx := context.Background()
	err = client.CreateService(ctx, svcInstance)
	if err != nil {
		log.Fatalf("Error creating Knative service: %s", err.Error())
	}

	log.Printf("Creating Prometheus support for service: %s", spec.Name)

	// wait for service be ready and forward payload
	go a.waitForServiceReadyAndForward(spec, requestPayloads)
}

// forwards the requests to existing service
func (a *Assigner) CurrentService(spec ServiceSpec, requestPayloads []io.ReadCloser) {
	// Forward each request one by one
	for _, requestPayload := range requestPayloads {
		go a.forwardRequest(spec.Name, requestPayload)
	}
}

// wait for service be ready and forward payload
func (a *Assigner) waitForServiceReadyAndForward(spec ServiceSpec, requestPayloads []io.ReadCloser) {
	log.Printf("Waiting for service to be ready - Name: %s", spec.Name)
	timeCounter := 0

	p := commands.KnParams{}
	p.Initialize()
	knClient, err := p.NewServingClient(namespace)
	if err != nil {
		log.Fatalf("Error creating Knative serving client: %s", err.Error())
	}

	ctx := context.Background()
	for {
		service, err := knClient.GetService(ctx, spec.Name)
		if err != nil {
			log.Fatalf("Error getting Knative service: %s", err.Error())
		}

		// wait for service to be ready
		for _, condition := range service.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				log.Printf("\nKnative Service is ready - Name: %s", spec.Name)
				// Forward each request payload of the request group one by one
				for _, requestPayload := range requestPayloads {
					a.forwardRequest(spec.Name, requestPayload)
				}
				return
			}
		}

		log.Printf("Waiting service %s to be ready ... (%d seconds waited)", spec.Name, timeCounter)
		timeCounter += 1
		time.Sleep(1 * time.Second)
	}
}

// forward a request to the service using kourier-internal with Host header and print request info
func (a *Assigner) forwardRequest(Name string, requestPayload io.ReadCloser) {
	log.Printf("Forwarding request to service: %s", Name)

	payload, err := ioutil.ReadAll(requestPayload)
	if err != nil {
		log.Printf("Error reading request payload: %s", err.Error())
		return
	}
	// Initialize Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster config: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Get the IP address of the kourier service
	kourierService, err := kubeClient.CoreV1().Services("kourier-system").Get(context.TODO(), "kourier", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Failed to get kourier service: %v", err)
	}

	kourierIP := kourierService.Status.LoadBalancer.Ingress[0].IP
	log.Printf("Kourier IP: %s", kourierIP)

	// Use Knative client to get the service URL
	p := commands.KnParams{}
	p.Initialize()
	knClient, err := p.NewServingClient(namespace)
	if err != nil {
		log.Fatalf("Error creating Knative serving client: %s", err.Error())
	}

	service, err := knClient.GetService(context.Background(), Name)
	if err != nil {
		log.Fatalf("Error getting Knative service: %s", err.Error())
	}

	serviceURL := service.Status.URL.String()
	log.Printf("Service URL: %s", serviceURL)
	// Remove "https://" prefix if it exists in the service URL
	if strings.HasPrefix(serviceURL, "http://") {
		serviceURL = strings.TrimPrefix(serviceURL, "http://")
	}

	// Set the Host header to Name.default.127.0.0.1.nip.io
	host := fmt.Sprintf(serviceURL)
	url := "http://" + kourierIP

	// Create the HTTP POST request
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Failed to create request: %v\n", err)
		return
	}

	// Set headers
	req.Host = host
	req.Header.Set("Content-Type", "application/json")

	// Print out the information (Host, URL, Headers, and Payload)
	log.Printf("Request Information:")
	log.Printf("URL: %s", url)
	log.Printf("Host: %s", host)
	log.Printf("Content-Type: %s", req.Header.Get("Content-Type"))
	log.Printf("Payload: %s", string(payload))

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to forward request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Process the response
	respPayload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response: %v\n", err)
		return
	}

	log.Printf("Response Payload from service: %s", string(respPayload))
	log.Println("Response sent back to original sender")
}
