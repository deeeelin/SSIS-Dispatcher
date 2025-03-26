package main

import (
	"fmt"
	"log"
	"net/http"
)

const maxRequestsPerGroup = 1 // Threshold to define how much request forms a group
const numWorkers = 1          // number of preprocess workers

func main() {

	// Start worker goroutines for precprocessing and grouping requests
	for i := 0; i < numWorkers; i++ {
		go worker()
	}
	// Start a single goroutine for the sequential processing of request groups
	go sequentialProcessor()

	// Start a Request Handler
	http.HandleFunc("/", handleRequest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handleRequest processes incoming HTTP requests and enqueues them to request channel
func handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Println("Received HTTP request:")
	// Print the incoming request information
	if r == nil {
		http.Error(w, "Request is nil", http.StatusBadRequest)
		return
	}
	log.Printf("Request Content: %v", r)

	// Parse and enqueue the request
	request := parseRequest(r)
	if request.Model == "" {
		// Respond with 400 Bad Request if parsing failed or request is empty
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	requestChannel <- request

	// Acknowledge receipt of the request
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Request received for model: %s", request.Model)
}

// worker processes requests from the requestChannel in parallel, groups them and do preprocessing
func worker() {
	preprocessor := Preprocessor{}
	for req := range requestChannel {
		model := req.Model

		mu.Lock()
		group := getOrCreateRequestGroup(model) // check if there is a forming group of this type of request
		addRequestToGroup(&group, req)          // add to the forming group
		modelGroups[model] = group

		if len(group.Requests) >= maxRequestsPerGroup { // check if forming group has enough request to form a complete group
			delete(modelGroups, model) // pop the group
			mu.Unlock()

			preprocessor.process(&group) //  preprocess the complete group
			processChannel <- group      // enqueue to prcessChannel for seqential processor
		} else {
			mu.Unlock()
		}
	}
}

// sequentialProcessor handles the Decide and Assign steps sequentially
func sequentialProcessor() {
	processor := Processor{}
	assigner := Assigner{}
	for group := range processChannel {
		log.Printf("Processing group sequentially for model: %s", group.Requests[0].Model)

		serviceSpec := processor.DecideService(group) // decide the service spec
		assigner.AssignService(serviceSpec, group)    // create the service and forward the request
	}
}
