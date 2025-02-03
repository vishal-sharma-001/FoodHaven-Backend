# Variables
PROJECT_NAME = foodhaven-449707
IMAGE_NAME = foodhaven
REPO_NAME = foodhaven
REGISTRY_LOCATION = us-central1-docker.pkg.dev
FULL_IMAGE_PATH = $(REGISTRY_LOCATION)/$(PROJECT_NAME)/$(REPO_NAME)/$(IMAGE_NAME)
TAG = latest

# Default target
all: clean build-ui build

# Build UI and Backend
build-ui:  
	@ cd $(FOODHAVEN_UI_REPO) && npm run build
	@ cd $(FOODHAVEN_BACKEND_REPO) && mkdir -p FoodHavenUI
	@ cp -rpf $(FOODHAVEN_UI_REPO)/build/* $(FOODHAVEN_BACKEND_REPO)/FoodHavenUI

build:
	@ go build
# Clean build artifacts
clean: 
	@rm -rf $(FOODHAVEN_UI_REPO)/build
	@rm -rf $(FOODHAVEN_BACKEND_REPO)/FoodHavenUI
	@rm -rf $(FOODHAVEN_BACKEND_REPO)/FoodHaven-Backend

# Build Docker Image
docker-build:
	@ echo "Building Docker image..."
	@ docker build --no-cache -t $(IMAGE_NAME) .

# Tag Docker Image
docker-tag: docker-build
	@ echo "Tagging Docker image..."
	@ docker tag $(IMAGE_NAME) $(FULL_IMAGE_PATH):$(TAG)

# Push Docker Image to Artifact Registry
docker-push: docker-tag
	@ echo "Pushing Docker image to Artifact Registry..."
	@ docker push $(FULL_IMAGE_PATH):$(TAG)
	@ echo "Docker image pushed successfully to $(FULL_IMAGE_PATH):$(TAG)."

push: all docker-push

# Run the application
run: clean build-ui
	@ go run main.go
