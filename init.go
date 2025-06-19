package main

import (
	"log"
	"os"
	"strings"
)

var ConfigList []string
var ConfigMap map[string]int
var namespace string
var image string

func init() {

	data, err := os.ReadFile("/etc/dispatcher-config/gpu-resource-config")
	if err != nil {
		log.Fatalf("Failed to read gpu resource config file: %v", err)
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ConfigList = append(ConfigList, line)
		ConfigMap[line] = 0
	}

	log.Printf("Configured GPU resource config: %v", ConfigList)

	data, err = os.ReadFile("/etc/dispatcher-config/service-namespace")
	if err != nil {
		log.Fatalf("Failed to read inference service namespace config: %v", err)
	}

	namespace = strings.TrimSpace(string(data))
	if namespace == "" {
		log.Fatalf("Namespace cannot be empty in the configuration")
	}

	log.Printf("Configured inference service namespace: %s", namespace)

	data, err = os.ReadFile("/etc/dispatcher-config/inference-image")
	if err != nil {
		log.Fatalf("Failed to read namespace config: %v", err)
	}

	image = strings.TrimSpace(string(data))
	if image == "" {
		log.Fatalf("Inference image cannot be empty in the configuration")
	}

	log.Printf("Configured Inference image: %s", image)
}
