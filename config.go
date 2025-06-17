package main

var testImage = map[string]string{
	"image":   "ghcr.io/deeeelin/knative-service:latest",
	"command": "",
}
var inferImage = map[string]string{
	"image":   "ghcr.io/llshang/tgi_custom:latest", //"ghcr.io/huggingface/text-generation-inference:1.4.4", // "ghcr.io/deeeelin/inference_server:latest"
	"command": "",
}

var gpuMode = "mig" // <PLACEHOLDER:Choose "mps or "mig" mode>

var namespace = "nthulab"
