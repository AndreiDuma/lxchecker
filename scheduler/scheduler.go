package main

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
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

func Submit(ctx context.Context, options SubmitOptions) (SubmitResponse, error) {
	r := SubmitResponse{}

	// TODO: make docker host configurable (ENV?)
	// TODO: move this outside of Submit
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, nil)
	if err != nil {
		return r, err
	}

	// create the container
	config := &container.Config{
		Image: options.Image,
	}
	container, err := cli.ContainerCreate(ctx, config, nil, nil, "")
	if err != nil {
		return r, err
	}

	// tar the submission
	tar, err := tarSubmission(options.Submission)
	if err != nil {
		return r, err
	}

	// copy submission to container
	copyOptions := types.CopyToContainerOptions{
		ContainerID: container.ID,
		Path:        "/submission/", // TODO: make this configurable
		Content:     tar,
	}
	if err = cli.CopyToContainer(ctx, copyOptions); err != nil {
		return r, err
	}

	// start the container
	if err = cli.ContainerStart(ctx, container.ID); err != nil {
		return r, err
	}

	// wait for the container to exit
	// TODO: add timeout here
	r.ExitCode, err = cli.ContainerWait(ctx, container.ID)
	if err != nil {
		return r, err
	}

	// get container logs
	logsOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}
	r.Logs, err = cli.ContainerLogs(ctx, container.ID, logsOptions)
	if err != nil {
		return r, err
	}
	return r, nil
}

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("submission")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	options := SubmitOptions{
		Image:      "so_tema3",
		Submission: fileBytes, // TODO: send this as an io.Reader
	}
	response, err := Submit(context.Background(), options)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	io.Copy(w, response.Logs)
}

func ResultHandler(w http.ResponseWriter, req *http.Request) {
}

func main() {
	http.HandleFunc("/submit", SubmitHandler)
	http.HandleFunc("/result", ResultHandler)
	panic(http.ListenAndServe(":8080", nil))

	// TODO: get the submission from client
	archivePath := "DUMA.zip"
	archive, err := ioutil.ReadFile(archivePath)
	if err != nil {
		panic(err)
	}

	response, err := Submit(context.Background(), SubmitOptions{
		Image:      "so_tema3",
		Submission: archive,
	})
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(os.Stdout, response.Logs)
	if err != nil && err != io.EOF {
		panic(err)
	}
}
