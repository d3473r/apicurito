FROM golang:1.26
WORKDIR /src

# Copy over the sources (including the dist/ folder with the built
# Angular application, which gets embedded into the binary).
COPY . /src
RUN CGO_ENABLED=0 go build -o /main .
