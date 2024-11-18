	#!/bin/bash

# Build the Docker image
docker image build -t forum .

# Run the Docker container
docker container run -p 8080:8080 --detach --name forum forum

#MSYS_NO_PATHCONV=1 docker exec -it forum /bin/bash