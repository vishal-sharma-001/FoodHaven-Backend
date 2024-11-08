FROM golang:1.23.1

LABEL description="FoodHaven Server"
COPY ./FoodHavenUI ./FoodHavenUI 
COPY ./FoodHaven-Backend ./FoodHaven-Backend

CMD ["./FoodHaven-Backend"]