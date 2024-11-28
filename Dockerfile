FROM golang:1.23.1

LABEL description="FoodHaven Server"
RUN ls
COPY ./FoodHavenUI ./FoodHavenUI 
COPY ./FoodHaven-Backend ./FoodHaven-Backend
COPY .env .env

CMD ["./FoodHaven-Backend"]