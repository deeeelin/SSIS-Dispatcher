# SSIS-Dispatcher

## About
The SSIS-Dispatcher project is a subproject branched from the SSIS(Scalable Serving Inference System for Language Models with NVIDIA MIG) project. It is a served as a serving manager component in the system. SSIS-Dispatcher is capable of receiving model inference requests and luanching inference pod under [Knative](https://knative.dev/docs/) framework while leveraging GPU sharing features supported my Nvidia [Multi-Instance GPU(MIG)](https://www.nvidia.com/en-us/technologies/multi-instance-gpu/) or [Multi-Process Service (MPS)](https://docs.nvidia.com/deploy/mps/index.html), which allows finegrained unitlization of GPU resources, enhancing system efficiency.
* Check out the [SSIS project repo](https://github.com/mike911209/KubeComp-MIG), for additional autoscaler or performance monitor support.

## Getting Start
### Prerequisite
* Requires a kubernetes cluster with version > 1.28
* This demo project default runs all knative service, pods  on `nthulab` namespace
* You should have MIG or MPS kubernetes resource registered on your cluster
    * For MIG environment setup, reference the [GPU operator documentation](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/gpu-operator-mig.html)
    * For MPS setup, recommended [Nebuly GPU device plugin](https://github.com/nebuly-ai/k8s-device-plugin)

### 1. Setup Knative and Kourier Ingress/ Load Balancer

* Run `make setup_knative`
* `k get po -n kourier-system`, check if kourier gateway is running
* `k get svc -n kourier-system`, check if kourier svc and kourier-internal service is established
* You can use `curl <kourier service external ip>` to test kourier external gateway or run a pod on cluster that runs `curl http://kourier-internal.kourier-system.svc.cluster.local` to check the in-cluster gateway is operating
* Use `kn service list` and find the url for the dispatcher, ex: `http://dispatcher.nthulab.192.168.1.10.sslip.io`

### 2. Build Your Own Dispatcher Image

* Run `make build`

### 3. Deploy dispatcher

* Run `make deploy`

### 4. Configure Dispatcher and Restart pod

* Run `kubectl edit configmap dispatcher-config
* Edit data section to set service namespace, inference image and GPU resource names that applies to your system environment
    * The MIG resource defined in node may have the example resource name format below:
    ```
    nvidia.com/mig-1g.5gb
    nvidia.com/mig-2g.10gb
    nvidia.com/mig-3g.20gb
    nvidia.com/mig-4g.20gb
    nvidia.com/mig-7g.40gb
    ```
    * The nebuly MPS resource defined in node may have the example the resource name format below:
    ```
    nvidia.com/gpu-1gb
    nvidia.com/gpu-2gb
    nvidia.com/gpu-3gb
    nvidia.com/gpu-4gb
    ...
    nvidia.com/gpu-30gb
    nvidia.com/gpu-31gb
    nvidia.com/gpu-32gb
    ```

* Restart the dispatcher pod to reload configurations (by deleting it)

### 5. Forward kourier in-cluster gateway 

* Assume the cluster external ip is unavailable, we make our test using in-cluster ip, which is likely available in most cases

* Open another terminal window , then : `make forward`

### 6. Send test API request to Dispatcher
* Export your HuggingFace token : `export HF_TOKEN="<Your token>"`
* Change Directory to `/test` and install required python package through `pip install -r requirements.txt`
* Run `python test.py` to send sample request to Dispatcher

### (OPTIONAL) Send Customize request to Dispatcher
* Make sure you done all steps above.
* You can set custom request through modifying `/test/payload.json`	:
```
{
    "token": "What is Deep Learning?",
    "par": {
        "max_new_tokens": "20"
    },
    "env": {
        "MODEL_ID": "openai-community/gpt2",
        "HF_TOKEN": ""
    }
}
``` 
* Reference for parameters (par): https://huggingface.co/docs/transformers/main_classes/text_generation
* Reference for environment variables (env) : https://huggingface.co/docs/text-generation-inference/main/en/reference/launcher

## Uninstall Project
* Delete all service running
* Run `make clean` to remove dispatcher
* Run `make remove_knative` to remove knative
