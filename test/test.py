from kubernetes import client, config
import json
import requests
import os

def get_ksvc_url(namespace, ksvc_name):
    # Load kubeconfig
    config.load_kube_config()

    # Create a custom objects API client
    api_instance = client.CustomObjectsApi()

    # Get the Knative Service (ksvc) details
    try:
        ksvc = api_instance.get_namespaced_custom_object(
            group="serving.knative.dev",
            version="v1",
            namespace=namespace,
            plural="services",
            name=ksvc_name
        )
        # Extract the URL from the status
        url = ksvc.get("status", {}).get("url", None)
        if url:
            return url
        else:
            raise ValueError("URL not found in Knative Service status.")
    except client.exceptions.ApiException as e:
        print(f"Exception when calling CustomObjectsApi: {e}")
        raise

def read_payload():
    with open("payload.json", "r") as file:
        data = json.load(file)
    
    data["env"]["HF_TOKEN"] = os.environ.get("HF_TOKEN")
    return data

def send_request(host, payload):
    print(f"payload : {payload}")
    headers = {
        "Content-Type": "application/json",
        "Host": str(host[7:]),
    }

    # Define the URL as local
    url = "http://localhost:8080"

    # Send the POST request
    response = requests.post(url, headers=headers, json=payload)

    # Print the response
    print(f"Response status code: {response.status_code}")
    print(f"Response body: {response.text}")


if __name__ == "__main__":
    namespace = "nthulab"  # Replace with your namespace
    ksvc_name = "dispatcher"  # Replace with your Knative Service name

    try:
        host_url = get_ksvc_url(namespace, ksvc_name)
        print(f"The URL for Knative Service '{ksvc_name}' is: {host_url}")
        payload = read_payload()
        print(f"Payload: {payload}")
        send_request(host_url, payload)
     
    except Exception as e:
        print(f"Error: {e}")

