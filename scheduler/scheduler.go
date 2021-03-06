package scheduler

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// makeSubmissionTar creates a tar archive from `submission`.
// The submission is place at `path` inside the archive.
func makeSubmissionTar(submission []byte, path string) (io.Reader, error) {
	buffer := new(bytes.Buffer)
	tw := tar.NewWriter(buffer)

	if err := tw.WriteHeader(&tar.Header{
		Name: path,
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

// SubmitOptions holds parameters for Submit.
type SubmitOptions struct {
	Image          string
	Submission     []byte
	SubmissionPath string
	Timeout        time.Duration
}

// SubmitResponse holds data returned from Submit.
type SubmitResponse struct {
	Logs     []byte
	ExitCode int
}

type Scheduler struct {
	cli *client.Client
}

// New creates a new scheduler object, encapsulating a connection to a Docker
// host.
func New() *Scheduler {
	var err error
	scheduler := &Scheduler{}
	if scheduler.cli, err = client.NewEnvClient(); err != nil {
		panic(err)
	}
	return scheduler
}

// Submit prepares a submission, creates a container for it, starts it, waits
// for it to exit and returns the logs.
func (scheduler *Scheduler) Submit(ctx context.Context, options SubmitOptions) (SubmitResponse, error) {
	r := SubmitResponse{}

	// pull the required image from the registry
	// TODO: figure out a way to make this faster
	reader, err := scheduler.cli.ImagePull(ctx, options.Image, types.ImagePullOptions{})
	if err != nil {
		return r, fmt.Errorf("Failed to pull image: %v", err)
	}
	// wait for the pull to finish
	if _, err := io.Copy(ioutil.Discard, reader); err != nil {
		return r, fmt.Errorf("Error while waiting for image pull to finish: %v", err)
	}
	reader.Close()

	// create the container
	config := &container.Config{
		Image: options.Image,
		Tty:   true,
	}
	container, err := scheduler.cli.ContainerCreate(ctx, config, nil, nil, "")
	if err != nil {
		return r, fmt.Errorf("Failed to create container: %v", err)
	}

	// tar the submission
	tar, err := makeSubmissionTar(options.Submission, options.SubmissionPath)
	if err != nil {
		return r, fmt.Errorf("Couldn't tar submission: %v", err)
	}

	// copy submission to container
	if err = scheduler.cli.CopyToContainer(ctx, container.ID, "/", tar, types.CopyToContainerOptions{}); err != nil {
		return r, fmt.Errorf("Failed to copy submission to container: %v", err)
	}

	// start the container
	if err = scheduler.cli.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); err != nil {
		return r, fmt.Errorf("Failed to start container: %v", err)
	}

	// wait for the container to exit
	ctxWait, cancel := context.WithTimeout(ctx, options.Timeout)
	r.ExitCode, err = scheduler.cli.ContainerWait(ctxWait, container.ID)
	cancel()
	if err != nil {
		return r, fmt.Errorf("Wait failed: %v", err)
	}

	// get container logs
	logsOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}
	logsReader, err := scheduler.cli.ContainerLogs(ctx, container.ID, logsOptions)
	if err != nil {
		return r, fmt.Errorf("Couldn't get logs from container: %v", err)
	}
	if r.Logs, err = ioutil.ReadAll(logsReader); err != nil {
		return r, fmt.Errorf("Failed reading the logs: %v", err)
	}
	return r, nil
}
