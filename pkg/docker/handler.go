// Package docker implements a tinyFaaS backend that runs each function as a
// set of Docker containers. For every function it builds a dedicated image,
// creates a dedicated network and one container per requested thread.
package docker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/moby/go-archive"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/OpenFogStack/tinyFaaS/pkg/manager"
	"github.com/OpenFogStack/tinyFaaS/pkg/util"
)

const (
	// TmpDir is the temporary folder where the build context of each function
	// (runtime files + function code) is assembled.
	TmpDir = "./tmp"

	// containerTimeout is the stop timeout in seconds.
	containerTimeout = 1
)

// dockerHandler represents one deployed function. It implements manager.Handler
// and keeps track of every Docker resource created for it (image, network,
// containers) so they can be started and cleaned up later.
type dockerHandler struct {
	name       string         // function name as given by the user
	env        string         // language runtime
	threads    int            // number of container replicas
	uniqueName string         // name + UUID, used for image, network and containers
	filePath   string         // temporary build folder
	client     *client.Client // Docker API client
	network    string         // ID of the Docker network created for this function
	containers []string       // IDs of the containers created for this function
	handlerIPs []string       // IPs of the running containers, filled by Start
}

// DockerBackend is the entry point of this package: it creates function
// handlers that run on the local Docker daemon.
type DockerBackend struct {
	client     *client.Client // Docker API client shared by all handlers
	tinyFaaSID string         // ID of this tinyFaaS instance, used as a resource label
}

// New creates a DockerBackend. The Docker client is configured from the
// environment (DOCKER_HOST etc.) and negotiates the API version with the
// daemon. It aborts the program if the client cannot be created.
func New(tinyFaaSID string) *DockerBackend {
	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("error creating docker client: %s", err)
		return nil
	}

	return &DockerBackend{client: client, tinyFaaSID: tinyFaaSID}
}

// Stop is a no-op: there is nothing to release at backend level, since
// resources are cleaned up per function in Destroy.
func (db *DockerBackend) Stop() error {
	return nil
}

// Create prepares a function for execution: it assembles the build context,
// builds the Docker image, creates a dedicated network and creates (but does
// not start) one container per thread. Call Start on the returned handler to
// actually run the function.
func (db *DockerBackend) Create(name string, env string, threads int, filedir string, envs map[string]string) (manager.Handler, error) {
	// Make a unique function name by appending uuid string to function name.
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	dh := &dockerHandler{
		name:       name,
		env:        env,
		client:     db.client,
		threads:    threads,
		containers: make([]string, 0, threads),
		handlerIPs: make([]string, 0, threads),
	}

	dh.uniqueName = name + "-" + uuid.String()
	log.Println("creating function", name, "with unique name", dh.uniqueName)

	// Make a folder for the function (mkdir <folder>).
	dh.filePath = path.Join(TmpDir, dh.uniqueName)
	err = os.MkdirAll(dh.filePath, 0777)
	if err != nil {
		return nil, err
	}

	// Copy Docker stuff into folder (cp runtimes/<env>/* <folder>). Note the
	// runtime files are embedded in the binary.
	err = util.CopyDirFromEmbed(runtimes, path.Join(runtimesDir, dh.env), dh.filePath)
	if err != nil {
		return nil, err
	}
	log.Println("copied runtime files to folder", dh.filePath)

	// Copy function into folder into a subfolder called fn (cp <file>
	// <folder>/fn).
	err = os.MkdirAll(path.Join(dh.filePath, "fn"), 0777)
	if err != nil {
		return nil, err
	}
	err = util.CopyAll(filedir, path.Join(dh.filePath, "fn"))
	if err != nil {
		return nil, err
	}

	// Build image. The folder is sent to the daemon as a tar archive (the build
	// context). Equivalent to docker build -t <image> <folder>.
	tar, err := archive.TarWithOptions(dh.filePath, &archive.TarOptions{})
	if err != nil {
		return nil, err
	}

	r, err := db.client.ImageBuild(
		context.Background(),
		tar,
		client.ImageBuildOptions{
			Tags:       []string{dh.uniqueName},
			Remove:     true, // remove intermediate containers after the build
			Dockerfile: "Dockerfile",
			// labels let us identify resources belonging to this function and
			// to this tinyFaaS instance
			Labels: map[string]string{
				"tinyfaas-function": dh.name,
				"tinyFaaS":          db.tinyFaaSID,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	// Stream the build output to the log until the build finishes.
	defer r.Body.Close()
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		log.Println(scanner.Text())
	}
	log.Println("built image", dh.uniqueName)

	// Create network, each function gets its own isolated network. Equivalent
	// to: docker network create <network>.
	network, err := db.client.NetworkCreate(
		context.Background(),
		dh.uniqueName,
		client.NetworkCreateOptions{
			Labels: map[string]string{
				"tinyfaas-function": dh.name,
				"tinyFaaS":          db.tinyFaaSID,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	dh.network = network.ID
	log.Println("created network", dh.uniqueName, "with id", network.ID)

	// Convert the environment variable map to the "KEY=VALUE" format expected
	// by Docker.
	e := make([]string, 0, len(envs))
	for k, v := range envs {
		e = append(e, fmt.Sprintf("%s=%s", k, v))
	}

	// Create containers. Equivalent to: docker run -d --network <network> --name <container> <image>.
	for i := 0; i < dh.threads; i++ {
		c, err := db.client.ContainerCreate(
			context.Background(),
			client.ContainerCreateOptions{
				Config: &container.Config{
					Image: dh.uniqueName,
					Labels: map[string]string{
						"tinyfaas-function": dh.name,
						"tinyFaaS":          db.tinyFaaSID,
					},
					Env: e,
				},
				HostConfig: &container.HostConfig{
					// Attach the container to the function's own network.
					NetworkMode: container.NetworkMode(dh.uniqueName),
				},
				// Container names are suffixed with the replica index
				Name: dh.uniqueName + fmt.Sprintf("-%d", i),
			},
		)
		if err != nil {
			return nil, err
		}
		log.Println("created container", c.ID)

		dh.containers = append(dh.containers, c.ID)
	}

	// Remove the build context folder, since it is no longer needed once the
	// image is built.
	err = os.RemoveAll(dh.filePath)
	if err != nil {
		return nil, err
	}
	log.Println("removed folder", dh.filePath)

	return dh, nil

}

// IPs returns the IP addresses of the function's containers. The list is
// populated by Start, so it is empty before the function has been started.
func (dh *dockerHandler) IPs() []string {
	return dh.handlerIPs
}

// Start launches all containers of the function, collects their IPs and waits
// until each of them reports it is healthy. It returns an error if a container
// does not become ready in time.
func (dh *dockerHandler) Start() error {
	log.Printf("dh: %+v", dh)

	// Start containers. Equivalent to: docker start <container>.
	wg := sync.WaitGroup{}
	for _, c := range dh.containers {
		wg.Add(1)
		go func(c string) {
			_, err := dh.client.ContainerStart(
				context.Background(),
				c,
				client.ContainerStartOptions{},
			)
			wg.Done()
			if err != nil {
				log.Printf("error starting container %s: %s", c, err)
				return
			}

			log.Println("started container", c)
		}(c)
	}
	wg.Wait()

	// Get container IPs from the function's dedicated network. Equivalent to: docker inspect <container>.
	for _, container := range dh.containers {
		c, err := dh.client.ContainerInspect(
			context.Background(),
			container,
			client.ContainerInspectOptions{},
		)
		if err != nil {
			return err
		}

		ip := c.Container.NetworkSettings.Networks[dh.uniqueName].IPAddress.String()
		dh.handlerIPs = append(dh.handlerIPs, ip)

		log.Println("got ip", ip, "for container", container)
	}

	// Wait for the containers to be ready.
	for i, ip := range dh.handlerIPs {
		log.Println("waiting for container", ip, "to be ready")
		maxRetries := 10
		for {
			maxRetries--
			if maxRetries == 0 {
				log.Printf("container %s (ip %s) not ready after 10 retries", dh.containers[i], ip)
				log.Printf("getting logs for container %s", dh.containers[i])
				logs, err := dh.getContainerLogs(dh.containers[i])

				if err != nil {
					return fmt.Errorf("container %s not ready after 10 retries, error encountered when getting logs %s", ip, err)
				}

				log.Println(logs)

				log.Printf("end of logs for container %s", dh.containers[i])

				return fmt.Errorf("container %s not ready after 10 retries", ip)
			}

			client := http.Client{
				Timeout: 3 * time.Second,
			}

			resp, err := client.Get("http://" + ip + ":8000/health")
			if err != nil {
				log.Println(err)
				log.Println("retrying in 1 second")
				time.Sleep(1 * time.Second)
				continue
			}
			resp.Body.Close()
			// HTTP 200 means the function runtime is up and ready
			if resp.StatusCode == http.StatusOK {
				log.Println("container", ip, "is ready")
				break
			}
			log.Println("container", ip, "is not ready yet, retrying in 1 second")
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

// Destroy removes everything created for the function: it stops and removes
// the containers (in parallel), then removes the network and the image.
func (dh *dockerHandler) Destroy() error {
	log.Println("destroying function", dh.name)
	log.Printf("dh: %+v", dh)

	wg := sync.WaitGroup{}
	log.Printf("stopping containers: %v", dh.containers)
	for _, c := range dh.containers {
		log.Println("removing container", c)

		wg.Add(1)
		go func(c string) {
			log.Println("stopping container", c)

			timeout := 1 // seconds

			// Equivalent to: docker stop <container>.
			_, err := dh.client.ContainerStop(
				context.Background(),
				c,
				client.ContainerStopOptions{
					Timeout: &timeout,
				},
			)
			if err != nil {
				log.Printf("error stopping container %s: %s", c, err)
			}

			log.Println("stopped container", c)

			// Equivalent to: docker rm <container>.
			_, err = dh.client.ContainerRemove(
				context.Background(),
				c,
				client.ContainerRemoveOptions{},
			)
			wg.Done()
			if err != nil {
				log.Printf("error removing container %s: %s", c, err)
			}
		}(c)

		log.Println("removed container", c)
	}

	// Wait for all containers to be stopped and removed; the network can only
	// be deleted once no container is attached to it
	wg.Wait()

	// Remove network. Equivalent to: docker network rm <network>.
	_, err := dh.client.NetworkRemove(
		context.Background(),
		dh.network,
		client.NetworkRemoveOptions{},
	)
	if err != nil {
		return err
	}

	log.Println("removed network", dh.network)

	// Remove image. Equivalent to: docker rmi <image>.
	_, err = dh.client.ImageRemove(
		context.Background(),
		dh.uniqueName,
		client.ImageRemoveOptions{},
	)

	if err != nil {
		return err
	}

	log.Println("removed image", dh.uniqueName)

	return nil
}

// getContainerLogs returns the stdout and stderr logs of a single container.
// Every line is prefixed with the function name and container ID so that logs
// from different containers can be told apart. Stdout lines come first,
// followed by stderr lines.
func (dh *dockerHandler) getContainerLogs(c string) (string, error) {
	logs := ""

	// Equivalent to: docker logs --timestamps <container>.
	l, err := dh.client.ContainerLogs(
		context.Background(),
		c,
		client.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
		},
	)
	if err != nil {
		return logs, err
	}

	var lstdout bytes.Buffer
	var lstderr bytes.Buffer

	// Docker multiplexes stdout and stderr in a single stream. StdCopy
	// demultiplexes them into two separate buffers.
	_, err = stdcopy.StdCopy(&lstdout, &lstderr, l)

	l.Close()

	if err != nil {
		return logs, err
	}

	// Add a prefix to each line: function=<function> handler=<handler> <line>.
	scanner := bufio.NewScanner(&lstdout)

	for scanner.Scan() {
		logs += fmt.Sprintf("function=%s handler=%s %s\n", dh.name, c, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return logs, err
	}

	// Same for stderr.
	scanner = bufio.NewScanner(&lstderr)

	for scanner.Scan() {
		logs += fmt.Sprintf("function=%s handler=%s %s\n", dh.name, c, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return logs, err
	}

	return logs, nil
}

// Logs returns the combined logs of all containers of the function as a single
// reader.
func (dh *dockerHandler) Logs() (io.Reader, error) {
	// Get container logs. Equivalent to: docker logs <container>.
	var logs bytes.Buffer

	for _, c := range dh.containers {
		l, err := dh.getContainerLogs(c)
		if err != nil {
			return nil, err
		}

		logs.WriteString(l)
	}

	return &logs, nil
}
