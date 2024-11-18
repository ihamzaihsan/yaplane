
 ## Docker Image & Container Setup

The Ascii-Art-Web can be conveniently containerised using Docker. Here are the steps to build a Docker image and run it in a container:

1. Build the Docker image:

   ```bash
   docker image build -t forum .
   ```

1. Verify that the image has been created:

   ```bash
   docker images
   ```

1. Run the image in a Docker container:

   ```bash
   docker container run -p 8080:8080 -d forum
   ```

1. Confirm that the container is running:

   ```bash
   docker ps -a
   ```

1. Get a shell to the running container:

   ```bash
   docker ps -a
   ```

1. To build the image and run the container with a script:

   ```bash
   chmod +x build-and-run.sh
   $ ./build-and-run.sh
   ```

1. Find the name of the Docker / Running Docker

   ```bash
   docker ps -a
   docker ps
   docker ps -a | findstr <your_search_term>
   ```

1. Stop the Docker

   ```bash
   docker stop <container ID or name> 
   ```

   1. Delete the Container and image

   ```bash
   docker rm <container ID or name> 
   docker rmi <Image name>
   ```