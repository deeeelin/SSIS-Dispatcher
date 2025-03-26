package main

var testImage = map[string]string{
	"image":   "ghcr.io/deeeelin/knative-service:latest",
	"command": "",
}
var inferImage = map[string]string{
	"image":   "ghcr.io/huggingface/text-generation-inference:2.2.0", // "ghcr.io/deeeelin/inference_server:latest"
	"command": "",
}

var gpuMode = "mig" // <PLACEHOLDER:Choose "mps or "mig" mode>

var namespace = "nthulab"
