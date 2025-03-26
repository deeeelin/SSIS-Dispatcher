package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// Single Request object
type Request struct {
	Model string                 // string field for model
	Token string                 `json:"token"` // string field for token
	Env   map[string]string      `json:"env"`   // map of strings for env
	Par   map[string]interface{} `json:"par"`   // slice of maps for par
	Label map[string]string      `json:"label"` // labels to add on pod
}
type RequestGroup struct {
	Requests []Request
}

var (
	requestChannel = make(chan Request, 100)       // Channel to receive incoming requests
	modelGroups    = make(map[string]RequestGroup) // Map to store requests grouped by model
	processChannel = make(chan RequestGroup, 100)  // Channel to send preprocessed groups to sequential processor
	mu             sync.Mutex                      // Mutex to synchronize access to the modelGroups map
)

// parseRequest extracts data from the JSON payload and returns a Request object
func parseRequest(r *http.Request) Request {
	defer r.Body.Close()
	var req Request

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		return Request{}
	}
	// Parse the JSON payload into a Request object
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error parsing JSON: %v", err)
		return Request{}
	}

	// Check if "MODEL_ID" exists in req.Env
	invalidPattern := regexp.MustCompile(`[^a-z0-9]+`)
	if MODEL_ID, ok := req.Env["MODEL_ID"]; ok {
		// Find the index of the last "/"
		if idx := strings.LastIndex(MODEL_ID, "/"); idx != -1 {
			req.Model = strings.ToLower(MODEL_ID[idx+1:]) // Extract substring after the last "/"
		} else {
			req.Model = strings.ToLower(MODEL_ID) // If no "/" is found, keep the original value
		}
		// Remove characters that do not match the regex pattern
		req.Model = invalidPattern.ReplaceAllString(req.Model, "")
		if req.Model == "" {
			log.Printf("Error: No valid model ID after applying regex")
			return Request{}
		}
	} else {
		log.Printf("Error: MODEL_ID not found in request")
		// Optionally return an empty Request or handle the case as needed
		return Request{}
	}

	log.Printf("Token: %s", req.Token)
	log.Printf("Model: %s", req.Model)
	log.Printf("Env: %v", req.Env)
	log.Printf("Par: %v", req.Par)
	log.Printf("SLO: %v", req.Label)

	return req
}

// getOrCreateRequestGroup retrieves or creates a new RequestGroup for a given model
func getOrCreateRequestGroup(model string) RequestGroup {
	group, exists := modelGroups[model]
	if !exists {
		group = RequestGroup{}
	}
	return group
}

// addRequestToGroup adds a request to a request group and updates its properties
func addRequestToGroup(group *RequestGroup, req Request) {
	group.Requests = append(group.Requests, req)
	// group.TokenSum += req.TokenSize
	// if req.SLO < group.MinSLO || group.MinSLO == 0 {
	// 	group.MinSLO = req.SLO
	// }
}
