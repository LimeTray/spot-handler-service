from  golang:1.17.3-alpine3.14

# update incase of change in kube version
RUN wget https://storage.googleapis.com/kubernetes-release/release/v1.10.12/bin/linux/amd64/kubectl
RUN chmod 755 ./kubectl
RUN cp ./kubectl /usr/bin/

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /main

EXPOSE 8080

CMD [ "/main" ]