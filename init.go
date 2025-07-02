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
var mpsActiveThreadPercentageMap map[string]string

func init() {

	ConfigList = make([]string, 0)
	ConfigMap = make(map[string]int)
	mpsActiveThreadPercentageMap = make(map[string]string)

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

	data, err = os.ReadFile("/etc/dispatcher-config/mps-active-thread-percentage-config")
	if err != nil {
		log.Fatalf("Failed to read mps active thread percentage config file: %v", err)
	}

	lines = strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, ":") {
			log.Printf("Skipping invalid mps active thread percentage config line: %q", line)
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		mpsActiveThreadPercentageMap[key] = value
	}

	log.Printf("Configured mps active thread percentage config: %v", mpsActiveThreadPercentageMap)

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
