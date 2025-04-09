# Variables
IMAGE_NAME=ssis-dispatcher
IMAGE_TAG=main #<PLACEHOLDER: YOUR GITHUB BRANCH NAME>
KUBE_CONFIG=configuration.yaml
GITHUB_USER=deeeelin #<PLACEHOLDER: YOUR GITHUB USERNAME>

.PHONY: all build push deploy clean

all: setup_knative build deploy forward test

setup_knative:
	kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.15.1/serving-crds.yaml
	kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.15.1/serving-core.yaml
	kubectl apply -f https://github.com/knative/net-kourier/releases/download/knative-v1.15.1/kourier.yaml
	kubectl patch configmap/config-network --namespace knative-serving --type merge  --patch '{"data":{"ingress-class":"kourier.ingress.networking.knative.dev"}}'
	kubectl --namespace kourier-system get service kourier
	kubectl get pods -n knative-serving
	kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.15.1/serving-default-domain.yaml
	brew install knative/client/kn
	brew tap knative-extensions/kn-plugins
	brew install func
	kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.15.0/eventing-crds.yaml
	kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.15.0/eventing-core.yaml
	kubectl get pods -n knative-eventing
	kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.15.0/in-memory-channel.yaml
	kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.15.0/mt-channel-broker.yaml

remove_knative:
	kubectl delete -f https://github.com/knative/serving/releases/download/knative-v1.15.1/serving-crds.yaml
	kubectl delete -f https://github.com/knative/serving/releases/download/knative-v1.15.1/serving-core.yaml
	kubectl delete -f https://github.com/knative/net-kourier/releases/download/knative-v1.15.1/kourier.yaml
	kubectl delete -f https://github.com/knative/serving/releases/download/knative-v1.15.1/serving-default-domain.yaml
	kubectl delete -f https://github.com/knative/eventing/releases/download/knative-v1.15.0/eventing-crds.yaml
	kubectl delete -f https://github.com/knative/eventing/releases/download/knative-v1.15.0/eventing-core.yaml
	kubectl delete -f https://github.com/knative/eventing/releases/download/knative-v1.15.0/in-memory-channel.yaml
	kubectl delete -f https://github.com/knative/eventing/releases/download/knative-v1.15.0/mt-channel-broker.yaml
	brew uninstall kn
	brew untap knative/client
	brew uninstall func
	brew untap knative-extensions/kn-plugins

build:
		docker build -t ghcr.io/$(GITHUB_USER)/$(IMAGE_NAME):$(IMAGE_TAG) --platform linux/amd64 --push .
deploy:
		kubectl apply -f $(KUBE_CONFIG)
clean:
		kubectl delete -f $(KUBE_CONFIG)
forward:
		kubectl port-forward --namespace kourier-system $(shell kubectl get pod -n kourier-system -l "app=3scale-kourier-gateway" --output=jsonpath="{.items[0].metadata.name}") 8080:8080 19000:9000 8443:8443

