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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"

	"github.com/OpenFogStack/tinyFaaS/pkg/manager"
	"github.com/OpenFogStack/tinyFaaS/pkg/util"
)

const (
	TmpDir           = "./tmp"
	containerTimeout = 1
)

type dockerHandler struct {
	name       string
	env        string
	threads    int
	uniqueName string
	filePath   string
	client     *client.Client
	network    string
	containers []string
	handlerIPs []string
}

type DockerBackend struct {
	client     *client.Client
	tinyFaaSID string
}

func New(tinyFaaSID string) *DockerBackend {
	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("error creating docker client: %s", err)
		return nil
	}

	return &DockerBackend{
		client:     client,
		tinyFaaSID: tinyFaaSID,
	}
}

func (db *DockerBackend) Stop() error {
	return nil
}

func (db *DockerBackend) Create(name string, env string, threads int, filedir string, envs map[string]string) (manager.Handler, error) {

	// make a unique function name by appending uuid string to function name
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

	// make a folder for the function
	// mkdir <folder>
	dh.filePath = path.Join(TmpDir, dh.uniqueName)

	err = os.MkdirAll(dh.filePath, 0777)
	if err != nil {
		return nil, err
	}

	// copy Docker stuff into folder
	// cp runtimes/<env>/* <folder>

	err = util.CopyDirFromEmbed(runtimes, path.Join(runtimesDir, dh.env), dh.filePath)
	if err != nil {
		return nil, err
	}

	log.Println("copied runtime files to folder", dh.filePath)

	// copy function into folder
	// into a subfolder called fn
	// cp <file> <folder>/fn
	err = os.MkdirAll(path.Join(dh.filePath, "fn"), 0777)
	if err != nil {
		return nil, err
	}

	err = util.CopyAll(filedir, path.Join(dh.filePath, "fn"))
	if err != nil {
		return nil, err
	}

	// build image
	// docker build -t <image> <folder>
	tar, err := archive.TarWithOptions(dh.filePath, &archive.TarOptions{})
	if err != nil {
		return nil, err
	}

	r, err := db.client.ImageBuild(
		context.Background(),
		tar,
		types.ImageBuildOptions{
			Tags:       []string{dh.uniqueName},
			Remove:     true,
			Dockerfile: "Dockerfile",
			Labels: map[string]string{
				"tinyfaas-function": dh.name,
				"tinyFaaS":          db.tinyFaaSID,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		log.Println(scanner.Text())
	}

	log.Println("built image", dh.uniqueName)

	// create network
	// docker network create <network>
	network, err := db.client.NetworkCreate(
		context.Background(),
		dh.uniqueName,
		network.CreateOptions{
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

	e := make([]string, 0, len(envs))

	for k, v := range envs {
		e = append(e, fmt.Sprintf("%s=%s", k, v))
	}

	// create containers
	// docker run -d --network <network> --name <container> <image>
	for i := 0; i < dh.threads; i++ {
		container, err := db.client.ContainerCreate(
			context.Background(),
			&container.Config{
				Image: dh.uniqueName,
				Labels: map[string]string{
					"tinyfaas-function": dh.name,
					"tinyFaaS":          db.tinyFaaSID,
				},
				Env: e,
			},
			&container.HostConfig{
				NetworkMode: container.NetworkMode(dh.uniqueName),
			},
			nil,
			nil,
			dh.uniqueName+fmt.Sprintf("-%d", i),
		)

		if err != nil {
			return nil, err
		}

		log.Println("created container", container.ID)

		dh.containers = append(dh.containers, container.ID)
	}

	// remove folder
	// rm -rf <folder>
	err = os.RemoveAll(dh.filePath)
	if err != nil {
		return nil, err
	}

	log.Println("removed folder", dh.filePath)

	return dh, nil

}

func (dh *dockerHandler) IPs() []string {
	return dh.handlerIPs
}

func (dh *dockerHandler) Start() error {
	log.Printf("dh: %+v", dh)

	// start containers
	// docker start <container>

	wg := sync.WaitGroup{}
	for _, c := range dh.containers {
		wg.Add(1)
		go func(c string) {
			err := dh.client.ContainerStart(
				context.Background(),
				c,
				container.StartOptions{},
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

	// get container IPs
	// docker inspect <container>
	for _, container := range dh.containers {
		c, err := dh.client.ContainerInspect(
			context.Background(),
			container,
		)
		if err != nil {
			return err
		}

		dh.handlerIPs = append(dh.handlerIPs, c.NetworkSettings.Networks[dh.uniqueName].IPAddress)

		log.Println("got ip", c.NetworkSettings.Networks[dh.uniqueName].IPAddress, "for container", container)
	}

	// wait for the containers to be ready
	// curl http://<container>:8000/ready
	for i, ip := range dh.handlerIPs {
		log.Println("waiting for container", ip, "to be ready")
		maxRetries := 10
		for {
			maxRetries--
			if maxRetries == 0 {
				// container did not start properly!
				// give people some logs to look at
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

			// timeout of 1 second
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

			err := dh.client.ContainerStop(
				context.Background(),
				c,
				container.StopOptions{
					Timeout: &timeout,
				},
			)
			if err != nil {
				log.Printf("error stopping container %s: %s", c, err)
			}

			log.Println("stopped container", c)

			err = dh.client.ContainerRemove(
				context.Background(),
				c,
				container.RemoveOptions{},
			)
			wg.Done()
			if err != nil {
				log.Printf("error removing container %s: %s", c, err)
			}
		}(c)

		log.Println("removed container", c)
	}
	wg.Wait()

	// remove network
	// docker network rm <network>
	err := dh.client.NetworkRemove(
		context.Background(),
		dh.network,
	)
	if err != nil {
		return err
	}

	log.Println("removed network", dh.network)

	// remove image
	// docker rmi <image>
	_, err = dh.client.ImageRemove(
		context.Background(),
		dh.uniqueName,
		image.RemoveOptions{},
	)

	if err != nil {
		return err
	}

	log.Println("removed image", dh.uniqueName)

	return nil
}

func (dh *dockerHandler) getContainerLogs(c string) (string, error) {
	logs := ""

	l, err := dh.client.ContainerLogs(
		context.Background(),
		c,
		container.LogsOptions{
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

	_, err = stdcopy.StdCopy(&lstdout, &lstderr, l)

	l.Close()

	if err != nil {
		return logs, err
	}

	// add a prefix to each line
	// function=<function> handler=<handler> <line>
	scanner := bufio.NewScanner(&lstdout)

	for scanner.Scan() {
		logs += fmt.Sprintf("function=%s handler=%s %s\n", dh.name, c, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return logs, err
	}

	// same for stderr
	scanner = bufio.NewScanner(&lstderr)

	for scanner.Scan() {
		logs += fmt.Sprintf("function=%s handler=%s %s\n", dh.name, c, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return logs, err
	}

	return logs, nil
}

func (dh *dockerHandler) Logs() (io.Reader, error) {
	// get container logs
	// docker logs <container>
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
