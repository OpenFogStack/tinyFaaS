package dockerkv

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	fred "git.tu-berlin.de/mcc-fred/fred/proto/client"
	"github.com/OpenFogStack/tinyFaaS/pkg/manager"
	"github.com/OpenFogStack/tinyFaaS/pkg/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/google/uuid"
)

const (
	TmpDir           = "./tmp"
	containerTimeout = 1
	fredRegistry     = "git.tu-berlin.de:5000/mcc-fred/fred"
	fredLibImage     = "alexandra:v0.2.14"
	fredLibPort      = 10000
)

type dockerKVHandler struct {
	name             string
	env              string
	threads          int
	uniqueName       string
	filePath         string
	client           *client.Client
	network          string
	networkName      string
	containers       []string
	handlerIPs       []string
	fredLibContainer string
	fredLibIP        string
	fredLibPort      string
	fredLibUser      string
	fredKeygroup     string
	tinyFaaSID       string
}

type DockerKVBackend struct {
	client     *client.Client
	fredHost   string
	fredClient fred.ClientClient
	caCertPath string
	caKeyPath  string
	tinyFaaSID string
}

func New(tinyFaaSID string, fredNodeID string, fredHost string, fredPort int, fredPeeringPort int, caCertPath string, caKeyPath string) *DockerKVBackend {

	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("error creating docker client: %s", err)
		return nil
	}

	// pull the fred lib image
	// docker pull <image>
	_, err = client.ImagePull(
		context.Background(),
		fmt.Sprintf("%s/%s", fredRegistry, fredLibImage),
		types.ImagePullOptions{},
	)

	if err != nil {
		log.Fatalf("error pulling fred lib image: %s", err)
		return nil
	}

	log.Print("pulled fred lib image")

	// certificate preparation
	certsDir, err := filepath.Abs(path.Join(TmpDir, uuid.NewString()))
	if err != nil {
		log.Fatalf("error creating certificates for fred client: %s", err)
	}

	err = os.MkdirAll(certsDir, 0777)
	if err != nil {
		log.Fatalf("error creating certificates for fred client: %s", err)
	}

	caCertPath, err = filepath.Abs(caCertPath)
	if err != nil {
		log.Fatalf("error creating certificates for fred client: %s", err)
	}

	caKeyPath, err = filepath.Abs(caKeyPath)
	if err != nil {
		log.Fatalf("error creating certificates for fred client: %s", err)
	}

	// create a client for FReD
	// generate certificates for self
	clientKeyPath, clientCertPath, err := createCert("tinyfaas", []net.IP{}, certsDir, caCertPath, caKeyPath)

	if err != nil {
		log.Fatalf("error creating certificates for fred client: %s", err)
	}

	fredHost = fmt.Sprintf("%s:%d", fredHost, fredPort)

	fredClient, err := createClient(fredHost, clientCertPath, clientKeyPath, []string{caCertPath})

	if err != nil {
		log.Fatalf("error creating fred client: %s", err)
	}

	return &DockerKVBackend{
		client:     client,
		fredHost:   fredHost,
		fredClient: fredClient,
		caCertPath: caCertPath,
		caKeyPath:  caKeyPath,
		tinyFaaSID: tinyFaaSID,
	}
}

func (db *DockerKVBackend) Create(name string, env string, threads int, filedir string) (manager.Handler, error) {

	// make a unique function name by appending uuid string to function name
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	dh := &dockerKVHandler{
		name:       name,
		env:        env,
		threads:    threads,
		client:     db.client,
		containers: make([]string, 0, threads),
		handlerIPs: make([]string, 0, threads),
		tinyFaaSID: db.tinyFaaSID,
	}

	dh.uniqueName = fmt.Sprintf("%s-%s", name, uuid.String())
	log.Println("creating function", name, "with unique name", dh.uniqueName)

	// make a folder for the function
	// mkdir <folder>
	dh.filePath = path.Join(TmpDir, dh.uniqueName)

	err = os.MkdirAll(dh.filePath, 0777)
	if err != nil {
		return nil, err
	}

	// copy Docker stuff into folder
	// cp ./runtimes/<env>/* <folder>
	err = util.CopyAll(path.Join("./runtimes", dh.env), dh.filePath)
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

	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		log.Println(scanner.Text())
	}

	log.Println("built image", dh.uniqueName)

	// remove folder
	// rm -rf <folder>
	err = os.RemoveAll(dh.filePath)
	if err != nil {
		log.Printf("error removing folder %s: %s", dh.filePath, err)
		return nil, err
	}

	log.Println("removed folder", dh.filePath)

	dh.networkName = fmt.Sprintf("%s-network", dh.uniqueName)

	// create network
	// docker network create <network>
	network, err := db.client.NetworkCreate(
		context.Background(),
		dh.networkName,
		types.NetworkCreate{
			CheckDuplicate: true,
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

	log.Println("created network", dh.networkName, "with id", network.ID)

	// start fred lib
	// TODO: generate certificates
	fredLibContainerName := fmt.Sprintf("%s-fred", dh.uniqueName)

	dh.fredLibUser = fmt.Sprintf("fred-%s", dh.uniqueName)

	certPath := path.Join(TmpDir, uuid.String())

	err = os.MkdirAll(certPath, 0777)

	if err != nil {
		log.Printf("error creating certificates for fredlib: %s", err)
		return nil, err
	}

	certPath, err = filepath.Abs(certPath)

	if err != nil {
		log.Printf("error creating certificates for fredlib: %s", err)
		return nil, err
	}

	keyPath, certPath, err := createCert(dh.fredLibUser, []net.IP{}, certPath, db.caCertPath, db.caKeyPath)

	if err != nil {
		log.Printf("error creating certificates for fredlib: %s", err)
		return nil, err
	}

	// get absolute paths
	keyPath, err = filepath.Abs(keyPath)
	if err != nil {
		log.Printf("error creating certificates for fredlib: %s", err)
		return nil, err
	}

	certPath, err = filepath.Abs(certPath)
	if err != nil {
		log.Printf("error creating certificates for fredlib: %s", err)
		return nil, err
	}

	// get address of local handler network gateway to get fred node IP
	n, err := db.client.NetworkInspect(context.Background(), dh.network, types.NetworkInspectOptions{})
	if err != nil {
		log.Printf("error getting network info: %s", err)
		return nil, err
	}

	fredNodeIP := n.IPAM.Config[0].Gateway

	dh.fredLibPort = fmt.Sprintf("%d", fredLibPort)

	_, fredPort, err := net.SplitHostPort(db.fredHost)

	if err != nil {
		log.Printf("error getting fred port: %s", err)
		return nil, err
	}

	c, err := db.client.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: fmt.Sprintf("%s/%s", fredRegistry, fredLibImage),
			Cmd: []string{
				"--address", "0.0.0.0:10000",
				"--lighthouse", fmt.Sprintf("%s:%s", fredNodeIP, fredPort),
				"--alexandra-key", "/cert/middleware.key",
				"--alexandra-cert", "/cert/middleware.crt",
				"--clients-key", "/cert/middleware.key",
				"--clients-cert", "/cert/middleware.crt",
				"--ca-cert", "/cert/ca.crt",
				"--clients-skip-verify",
				"--alexandra-insecure",
			},
			Labels: map[string]string{
				"tinyFaaS":          db.tinyFaaSID,
				"tinyfaas-function": dh.name,
			},
		},
		&container.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/cert/middleware.crt", certPath),
				fmt.Sprintf("%s:/cert/middleware.key", keyPath),
				fmt.Sprintf("%s:/cert/ca.crt", db.caCertPath),
			},
			// PortBindings: nat.PortMap{
			// 	"10000/tcp": []nat.PortBinding{
			// 		{
			// 			HostIP:   "",
			// 			HostPort: fmt.Sprintf("%d", fredLibPort),
			// 		},
			// 	},
			// },
			NetworkMode: container.NetworkMode(dh.networkName),
		},
		nil,
		nil,
		fredLibContainerName,
	)

	if err != nil {
		log.Printf("error creating fred lib container: %s", err)
		return nil, err
	}

	log.Println("created fred lib container", c.ID)

	dh.fredLibContainer = c.ID

	// create fred keygroup
	dh.fredKeygroup = fmt.Sprintf("tinyFaaS%s%s", dh.name, uuid.String()[:8])
	err = db.createKeygroup(dh.fredKeygroup)

	if err != nil {
		log.Printf("error creating fred keygroup: %s", err)
		return nil, err
	}

	log.Println("created fred keygroup", dh.fredKeygroup)

	// add user to fred keygroup
	err = db.addUserToKeygroup(dh.fredKeygroup, dh.fredLibUser)

	if err != nil {
		log.Printf("error adding user to fred keygroup: %s", err)
		return nil, err
	}

	return dh, nil

}

func (db *DockerKVBackend) Stop() error {
	return nil
}

func (dh *dockerKVHandler) IPs() []string {
	return dh.handlerIPs
}

func (dh *dockerKVHandler) Start() error {
	log.Printf("dh: %+v", dh)

	// start fred lib
	err := dh.client.ContainerStart(
		context.Background(),
		dh.fredLibContainer,
		types.ContainerStartOptions{},
	)

	if err != nil {
		log.Printf("error starting fred lib %s: %s", dh.fredLibContainer, err)
		return err
	}

	log.Println("started fred lib", dh.fredLibContainer)

	// get fred lib IP
	c, err := dh.client.ContainerInspect(
		context.Background(),
		dh.fredLibContainer,
	)

	if err != nil {
		return err
	}

	log.Printf("getting fred lib ip for container %s", dh.fredLibContainer)

	log.Printf("networks: %+v", c.NetworkSettings.Networks[dh.networkName])

	dh.fredLibIP = c.NetworkSettings.Networks[dh.networkName].IPAddress

	// create containers
	// docker run -d --network <network> --name <container> <image>
	for i := 0; i < dh.threads; i++ {
		c, err := dh.client.ContainerCreate(
			context.Background(),
			&container.Config{
				Image: dh.uniqueName,
				Labels: map[string]string{
					"tinyfaas-function": dh.name,
					"tinyFaaS":          dh.tinyFaaSID,
				},
				Env: []string{
					fmt.Sprintf("__KV_KEYGROUP=%s", dh.fredKeygroup),
					fmt.Sprintf("__KV_HOST=%s", fmt.Sprintf("%s:%s", dh.fredLibIP, dh.fredLibPort)),
				},
			},
			&container.HostConfig{
				NetworkMode: container.NetworkMode(dh.networkName),
			},
			nil,
			nil,
			fmt.Sprintf("%s-%d", dh.uniqueName, i),
		)

		if err != nil {
			return err
		}

		log.Println("created function handler container", c.ID)

		dh.containers = append(dh.containers, c.ID)
	}

	// start containers
	// docker start <container>

	wg := sync.WaitGroup{}
	for _, container := range dh.containers {
		wg.Add(1)
		go func(c string) {
			err := dh.client.ContainerStart(
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
	for _, container := range dh.containers {
		c, err := dh.client.ContainerInspect(
			context.Background(),
			container,
		)
		if err != nil {
			return err
		}

		dh.handlerIPs = append(dh.handlerIPs, c.NetworkSettings.Networks[dh.networkName].IPAddress)

		log.Println("got ip", c.NetworkSettings.Networks[dh.networkName].IPAddress, "for container", container)
	}

	// wait for the containers to be ready
	// curl http://<container>:8000/ready
	for _, ip := range dh.handlerIPs {
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

			resp, err := client.Get(fmt.Sprintf("http://%s:8000/health", ip))
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

func (dh *dockerKVHandler) Destroy() error {
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

	// stop fred lib
	log.Println("stopping fred lib", dh.fredLibContainer)
	timeout := 1 // seconds

	err := dh.client.ContainerStop(
		context.Background(),
		dh.fredLibContainer,
		container.StopOptions{
			Timeout: &timeout,
		},
	)

	if err != nil {
		log.Printf("error stopping fred lib container: %s", err)
	}

	log.Println("stopped fredlib container")

	err = dh.client.ContainerRemove(
		context.Background(),
		dh.fredLibContainer,
		types.ContainerRemoveOptions{},
	)

	if err != nil {
		log.Printf("error removing fred lib container: %s", err)
	}

	log.Println("removed fred lib container")

	// remove network
	// docker network rm <network>
	err = dh.client.NetworkRemove(
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
		types.ImageRemoveOptions{},
	)

	if err != nil {
		return err
	}

	log.Println("removed image", dh.uniqueName)

	return nil
}

func (dh *dockerKVHandler) Logs() (string, error) {
	// get container logs
	// docker logs <container>
	var logs string
	for _, container := range dh.containers {
		l, err := dh.client.ContainerLogs(
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
