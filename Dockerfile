FROM golang:1.23.1

LABEL description="FoodHaven Server"
# Copy frontend and backend into the container
COPY ./FoodHavenUI ./FoodHavenUI
COPY ./FoodHaven-Backend ./FoodHaven-Backend

# Command to run the backend
CMD ["./FoodHaven-Backend"]