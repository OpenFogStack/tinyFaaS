package docker

import (
	"bufio"
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
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/google/uuid"
	"github.com/pfandzelter/tinyFaaS/pkg/util"
)

const (
	TmpDir           = "./tmp"
	containerTimeout = 1
)

type dockerHandler struct {
	functionName       string
	functionEnv        string
	functionThreads    int
	functionUniqueName string
	filePath           string
	thisNetwork        string
	thisContainers     []string
	thisHandlerIPs     []string
}

func Create(name string, env string, threads int, filedir string, tinyFaaSID string) (*dockerHandler, error) {

	// make a unique function name by appending uuid string to function name
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	dh := &dockerHandler{
		functionName:    name,
		functionEnv:     env,
		functionThreads: threads,
		thisContainers:  make([]string, 0, threads),
		thisHandlerIPs:  make([]string, 0, threads),
	}

	dh.functionUniqueName = name + "-" + uuid.String()
	log.Println("creating function", name, "with unique name", dh.functionUniqueName)

	// make a folder for the function
	// mkdir <folder>
	dh.filePath = path.Join(TmpDir, dh.functionUniqueName)

	err = os.MkdirAll(dh.filePath, 0777)
	if err != nil {
		return nil, err
	}

	// copy Docker stuff into folder
	// cp ./runtimes/<env>/* <folder>
	err = util.CopyAll(path.Join("./runtimes", dh.functionEnv), dh.filePath)
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

	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	// build image
	// docker build -t <image> <folder>
	tar, err := archive.TarWithOptions(dh.filePath, &archive.TarOptions{})
	if err != nil {
		return nil, err
	}

	r, err := client.ImageBuild(
		context.Background(),
		tar,
		types.ImageBuildOptions{
			Tags:       []string{dh.functionUniqueName},
			Remove:     true,
			Dockerfile: "Dockerfile",
			Labels: map[string]string{
				"tinyfaas-function": dh.functionName,
				"tinyFaaS":          tinyFaaSID,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		log.Println(scanner.Text())
	}

	log.Println("built image", dh.functionUniqueName)

	// create network
	// docker network create <network>
	network, err := client.NetworkCreate(
		context.Background(),
		dh.functionUniqueName,
		types.NetworkCreate{
			CheckDuplicate: true,
			Labels: map[string]string{
				"tinyfaas-function": dh.functionName,
				"tinyFaaS":          tinyFaaSID,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	dh.thisNetwork = network.ID

	log.Println("created network", dh.functionUniqueName, "with id", network.ID)

	// create containers
	// docker run -d --network <network> --name <container> <image>
	for i := 0; i < dh.functionThreads; i++ {
		container, err := client.ContainerCreate(
			context.Background(),
			&container.Config{
				Image: dh.functionUniqueName,
				Labels: map[string]string{
					"tinyfaas-function": dh.functionName,
					"tinyFaaS":          tinyFaaSID,
				},
			},
			&container.HostConfig{
				NetworkMode: container.NetworkMode(dh.functionUniqueName),
			},
			nil,
			nil,
			dh.functionUniqueName+fmt.Sprintf("-%d", i),
		)

		if err != nil {
			return nil, err
		}

		log.Println("created container", container.ID)

		dh.thisContainers = append(dh.thisContainers, container.ID)
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
	return dh.thisHandlerIPs
}

func (dh *dockerHandler) Start() error {
	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	log.Printf("dh: %+v", dh)

	// start containers
	// docker start <container>

	wg := sync.WaitGroup{}
	for _, container := range dh.thisContainers {
		wg.Add(1)
		go func(c string) {
			err := client.ContainerStart(
				context.Background(),
				c,
				types.ContainerStartOptions{},
			)
			wg.Done()
			if err != nil {
				log.Printf("error starting container %s: %s", c, err)
				return
			}

			log.Println("started container", c)
		}(container)
	}
	wg.Wait()

	// get container IPs
	// docker inspect <container>
	for _, container := range dh.thisContainers {
		c, err := client.ContainerInspect(
			context.Background(),
			container,
		)
		if err != nil {
			return err
		}

		dh.thisHandlerIPs = append(dh.thisHandlerIPs, c.NetworkSettings.Networks[dh.functionUniqueName].IPAddress)

		log.Println("got ip", c.NetworkSettings.Networks[dh.functionUniqueName].IPAddress, "for container", container)
	}

	// wait for the containers to be ready
	// curl http://<container>:8000/ready
	for _, ip := range dh.thisHandlerIPs {
		log.Println("waiting for container", ip, "to be ready")
		maxRetries := 10
		for {
			maxRetries--
			if maxRetries == 0 {
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
	log.Println("destroying function", dh.functionName)

	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("error creating docker client: %s", err)
		return err
	}

	log.Printf("fh: %+v", dh)

	wg := sync.WaitGroup{}
	log.Printf("stopping containers: %v", dh.thisContainers)
	for _, c := range dh.thisContainers {
		log.Println("removing container", c)

		wg.Add(1)
		go func(c string) {
			log.Println("stopping container", c)

			timeout := 1 // seconds

			err := client.ContainerStop(
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

			err = client.ContainerRemove(
				context.Background(),
				c,
				types.ContainerRemoveOptions{},
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
	err = client.NetworkRemove(
		context.Background(),
		dh.thisNetwork,
	)
	if err != nil {
		return err
	}

	log.Println("removed network", dh.thisNetwork)

	// remove image
	// docker rmi <image>
	_, err = client.ImageRemove(
		context.Background(),
		dh.functionUniqueName,
		types.ImageRemoveOptions{},
	)

	if err != nil {
		return err
	}

	log.Println("removed image", dh.functionUniqueName)

	return nil
}

func (dh *dockerHandler) Logs() (string, error) {
	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return "", err
	}

	// get container logs
	// docker logs <container>
	var logs string
	for _, container := range dh.thisContainers {
		l, err := client.ContainerLogs(
			context.Background(),
			container,
			types.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
			},
		)
		if err != nil {
			return "", err
		}

		lstr, err := io.ReadAll(l)
		l.Close()

		if err != nil {
			return "", err
		}

		logs += string(lstr)
		logs += "\n"
	}

	return logs, nil
}
