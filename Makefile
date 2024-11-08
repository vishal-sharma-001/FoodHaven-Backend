
IMAGE_NAME = foodhaven-image
TAG = v1.0

all: clean build-ui


build-ui:  
	@ cd $(FOODHAVEN_UI_REPO) && npm run build
	@ cd $(FOODHAVEN_BACKEND_REPO) && mkdir FoodHavenUI
	@ cp -rpf $(FOODHAVEN_UI_REPO)/build/* $(FOODHAVEN_BACKEND_REPO)/FoodHavenUI


clean: 
	@rm -rf $(FOODHAVEN_UI_REPO)/build
	@rm -rf $(FOODHAVEN_BACKEND_REPO)/FoodHavenUI


push: all
	@ docker build -t $(IMAGE_NAME):$(TAG) .


run: clean build-ui
	go run main.go


