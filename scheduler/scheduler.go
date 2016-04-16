package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"golang.org/x/net/context"
)

func tarSubmission(submission []byte) (io.Reader, error) {
	buffer := new(bytes.Buffer)
	tw := tar.NewWriter(buffer)

	if err := tw.WriteHeader(&tar.Header{
		Name: "submission.zip", // TODO: make this configurable
		Mode: 0444,
		Size: int64(len(submission)),
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(submission); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buffer, nil
}

type SubmitOptions struct {
	Image      string
	Submission []byte
}

type SubmitResponse struct {
	Logs     io.Reader
	ExitCode int
}

var (
	docker *client.Client
)

func Submit(ctx context.Context, options SubmitOptions) (SubmitResponse, error) {
	r := SubmitResponse{}

	// pull the required image from the registry
	reader, err := docker.ImagePull(ctx, options.Image, types.ImagePullOptions{})
	if err != nil {
		return r, fmt.Errorf("Failed to pull image: %v", err)
	}
	// wait for the pull to finish
	io.Copy(ioutil.Discard, reader)
	reader.Close()

	// create the container
	config := &container.Config{
		Image: options.Image,
	}
	container, err := docker.ContainerCreate(ctx, config, nil, nil, "")
	if err != nil {
		return r, fmt.Errorf("Failed to create container: %v", err)
	}

	// tar the submission
	tar, err := tarSubmission(options.Submission)
	if err != nil {
		return r, fmt.Errorf("Couldn't tar submission: %v", err)
	}

	// copy submission to container
	copyOptions := types.CopyToContainerOptions{
		ContainerID: container.ID,
		Path:        "/submission/", // TODO: make this configurable
		Content:     tar,
	}
	if err = docker.CopyToContainer(ctx, copyOptions); err != nil {
		return r, fmt.Errorf("Failed to copy submission to container: %v", err)
	}

	// start the container
	if err = docker.ContainerStart(ctx, container.ID); err != nil {
		return r, fmt.Errorf("Failed to start container: %v", err)
	}

	// wait for the container to exit
	// TODO: add timeout here
	r.ExitCode, err = docker.ContainerWait(ctx, container.ID)
	if err != nil {
		return r, fmt.Errorf("Wait failed: %v", err)
	}

	// get container logs
	logsOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}
	r.Logs, err = docker.ContainerLogs(ctx, container.ID, logsOptions)
	if err != nil {
		return r, fmt.Errorf("Couldn't get logs from container: %v", err)
	}
	return r, nil
}

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("submission")
	if err != nil {
		http.Error(w, "missing required `submission` field", http.StatusBadRequest)
		return
	}
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error while processing the uploaded file", http.StatusInternalServerError)
		return
	}

	options := SubmitOptions{
		Image:      "andreiduma/lxchecker_so_tema3",
		Submission: fileBytes, // TODO: send this as an io.Reader
	}
	response, err := Submit(context.Background(), options)
	if err != nil {
		log.Println(err)
		http.Error(w, "Submission could not be tested", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(w, response.Logs)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error while returning the logs", http.StatusInternalServerError)
	}
}

func ResultHandler(w http.ResponseWriter, req *http.Request) {
	// TODO
}

func main() {
	/*
		if os.Getenv("DOCKER_HOST") == "" || os.Getenv("DOCKER_API_VERSION") == "" {
			log.Fatalln("DOCKER_HOST and DOCKER_API_VERSION environment variables need to be set")
		}
	*/

	var err error
	docker, err = client.NewEnvClient()
	if err != nil {
		log.Fatalf("couldn't create Docker client: %v\n", err)
	}

	http.HandleFunc("/submit", SubmitHandler)
	http.HandleFunc("/result", ResultHandler)

	host := os.Getenv("LXCHECKER_SCHEDULER_HOST")
	if host == "" {
		host = ":5000"
	}
	log.Printf("Scheduler listening on %s...\n", host)
	log.Fatalln(http.ListenAndServe(host, nil))
}
