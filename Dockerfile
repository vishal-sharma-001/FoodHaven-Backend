FROM golang:1.23.1

LABEL description="FoodHaven Server"
# Copy frontend and backend into the container
COPY ./FoodHavenUI ./FoodHavenUI
COPY ./FoodHaven-Backend ./FoodHaven-Backend

# Install OpenSSL to generate the session key
RUN apt-get update && apt-get install -y openssl

ENV SESSION_KEY="e5f8d2c4b7a9e3f6a1b4c7d8e9f0a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9"

# Command to run the backend
CMD ["./FoodHaven-Backend"]