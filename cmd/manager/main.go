package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/google/uuid"
)

const (
	ConfigPort          = 8080
	RProxyConfigPort    = 8081
	RProxyListenAddress = ""
	RProxyBin           = "./rproxy"
	TmpDir              = "./tmp"
)

type ManagementService struct {
	id                    string
	functionHandlers      map[string]*FunctionHandler
	functionHandlersMutex sync.Mutex
	rproxy                *os.Process
	rproxyListenAddress   string
	rproxyConfigPort      int
	rproxyPort            map[string]int
}

type FunctionHandler struct {
	functionName       string
	functionEnv        string
	functionThreads    int
	functionUniqueName string
	filePath           string
	thisNetwork        string
	thisContainers     []string
	thisHandlerIPs     []string
}

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("manager: ")

	ms := &ManagementService{
		id:               uuid.New().String(),
		functionHandlers: make(map[string]*FunctionHandler),
	}

	ports := map[string]int{
		"coap": 5683,
		"http": 8000,
		"grpc": 9000,
	}

	for p := range ports {
		portstr := os.Getenv(p + "_PORT")

		if portstr == "" {
			continue
		}

		port, err := strconv.Atoi(portstr)

		if err != nil {
			log.Fatalf("invalid port for protocol %s: %s (must be an integer!)", p, err)
		}

		if port < 0 {
			delete(ports, p)
			continue
		}

		if port > 65535 {
			log.Fatalf("invalid port for protocol %s: %s (must be an integer lower than 65535!)", p, err)
		}

		ports[p] = port
	}

	ms.start(ConfigPort, ports)
}

func (ms *ManagementService) start(configurationPort int, rproxyPorts map[string]int) {
	// create rproxy
	log.Println("starting rproxy")
	ms.startRProxy(RProxyListenAddress, RProxyConfigPort, rproxyPorts)

	// create handlers
	r := http.NewServeMux()
	r.HandleFunc("/upload", ms.uploadHandler)
	r.HandleFunc("/delete", ms.deleteHandler)
	r.HandleFunc("/list", ms.listHandler)
	r.HandleFunc("/wipe", ms.wipeHandler)
	r.HandleFunc("/logs", ms.logsHandler)
	r.HandleFunc("/uploadURL", ms.urlUploadHandler)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c

		log.Println("received interrupt")
		log.Println("shutting down")

		// stop rproxy
		log.Println("stopping rproxy")
		err := ms.rproxy.Kill()

		if err != nil {
			log.Println(err)
		}

		// stop handlers
		log.Println("stopping handlers")
		err = ms.wipe()

		if err != nil {
			log.Println(err)
		}

		os.Exit(0)
	}()

	// start server
	log.Println("starting HTTP server")
	addr := fmt.Sprintf(":%d", ConfigPort)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatal(err)
	}
}

func (ms *ManagementService) startRProxy(rproxyListenAddress string, rproxyPort int, ports map[string]int) {
	// start rproxy
	log.Println("starting rproxy")

	ms.rproxyListenAddress = rproxyListenAddress
	ms.rproxyPort = ports
	ms.rproxyConfigPort = rproxyPort

	rproxyArgs := []string{fmt.Sprintf("%s:%d", rproxyListenAddress, rproxyPort)}

	for prot, port := range ports {
		rproxyArgs = append(rproxyArgs, fmt.Sprintf("%s:%s:%d", prot, rproxyListenAddress, port))
	}

	log.Println("rproxy args:", rproxyArgs)
	c := exec.Command(RProxyBin, rproxyArgs...)

	stdout, err := c.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	err = c.Start()
	if err != nil {
		log.Fatal(err)
	}

	ms.rproxy = c.Process

	log.Println("started rproxy")
}

func create(name string, env string, threads int, filedir string, tinyFaaSID string) (*FunctionHandler, error) {

	// make a unique function name by appending uuid string to function name
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	fh := &FunctionHandler{
		functionName:    name,
		functionEnv:     env,
		functionThreads: threads,
		thisContainers:  make([]string, 0, threads),
		thisHandlerIPs:  make([]string, 0, threads),
	}

	fh.functionUniqueName = name + "-" + uuid.String()
	log.Println("creating function", name, "with unique name", fh.functionUniqueName)

	// make a folder for the function
	// mkdir <folder>
	fh.filePath = path.Join(TmpDir, fh.functionUniqueName)

	err = os.MkdirAll(fh.filePath, 0777)
	if err != nil {
		return nil, err
	}

	// copy Docker stuff into folder
	// cp ./runtimes/<env>/* <folder>
	err = copyAll(path.Join("./runtimes", fh.functionEnv), fh.filePath)
	if err != nil {
		return nil, err
	}
	log.Println("copied runtime files to folder", fh.filePath)

	// copy function into folder
	// into a subfolder called fn
	// cp <file> <folder>/fn
	err = os.MkdirAll(path.Join(fh.filePath, "fn"), 0777)
	if err != nil {
		return nil, err
	}

	err = copyAll(filedir, path.Join(fh.filePath, "fn"))
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
	tar, err := archive.TarWithOptions(fh.filePath, &archive.TarOptions{})
	if err != nil {
		return nil, err
	}

	r, err := client.ImageBuild(
		context.Background(),
		tar,
		types.ImageBuildOptions{
			Tags:       []string{fh.functionUniqueName},
			Remove:     true,
			Dockerfile: "Dockerfile",
			Labels: map[string]string{
				"tinyfaas-function": fh.functionName,
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

	log.Println("built image", fh.functionUniqueName)

	// create network
	// docker network create <network>
	network, err := client.NetworkCreate(
		context.Background(),
		fh.functionUniqueName,
		types.NetworkCreate{
			CheckDuplicate: true,
			Labels: map[string]string{
				"tinyfaas-function": fh.functionName,
				"tinyFaaS":          tinyFaaSID,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	fh.thisNetwork = network.ID

	log.Println("created network", fh.functionUniqueName, "with id", network.ID)

	// create containers
	// docker run -d --network <network> --name <container> <image>
	for i := 0; i < fh.functionThreads; i++ {
		container, err := client.ContainerCreate(
			context.Background(),
			&container.Config{
				Image: fh.functionUniqueName,
				Labels: map[string]string{
					"tinyfaas-function": fh.functionName,
					"tinyFaaS":          tinyFaaSID,
				},
			},
			&container.HostConfig{
				NetworkMode: container.NetworkMode(fh.functionUniqueName),
			},
			nil,
			nil,
			fh.functionUniqueName+fmt.Sprintf("-%d", i),
		)

		if err != nil {
			return nil, err
		}

		log.Println("created container", container.ID)

		fh.thisContainers = append(fh.thisContainers, container.ID)
	}

	// remove folder
	// rm -rf <folder>
	err = os.RemoveAll(fh.filePath)
	if err != nil {
		return nil, err
	}

	log.Println("removed folder", fh.filePath)

	return fh, nil

}

func (fh *FunctionHandler) start() error {
	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	log.Printf("fh: %+v", fh)

	// start containers
	// docker start <container>
	for _, container := range fh.thisContainers {
		err := client.ContainerStart(
			context.Background(),
			container,
			types.ContainerStartOptions{},
		)
		if err != nil {
			return err
		}

		log.Println("started container", container)
	}

	// get container IPs
	// docker inspect <container>
	for _, container := range fh.thisContainers {
		c, err := client.ContainerInspect(
			context.Background(),
			container,
		)
		if err != nil {
			return err
		}

		fh.thisHandlerIPs = append(fh.thisHandlerIPs, c.NetworkSettings.Networks[fh.functionUniqueName].IPAddress)

		log.Println("got ip", c.NetworkSettings.Networks[fh.functionUniqueName].IPAddress, "for container", container)
	}

	// wait for the containers to be ready
	// curl http://<container>:8000/ready
	for _, ip := range fh.thisHandlerIPs {
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

func (fh *FunctionHandler) destroy() error {
	log.Println("destroying function", fh.functionName)

	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("error creating docker client: %s", err)
		return err
	}

	log.Printf("fh: %+v", fh)

	// stop containers
	// docker stop <container>
	log.Printf("stopping containers: %v", fh.thisContainers)
	for _, c := range fh.thisContainers {
		log.Println("stopping container", c)

		err := client.ContainerStop(
			context.Background(),
			c,
			container.StopOptions{},
		)
		if err != nil {
			return err
		}

		log.Println("stopped container", c)
	}

	// remove containers
	// docker rm <container>
	for _, container := range fh.thisContainers {
		log.Println("removing container", container)

		err := client.ContainerRemove(
			context.Background(),
			container,
			types.ContainerRemoveOptions{},
		)
		if err != nil {
			return err
		}

		log.Println("removed container", container)
	}

	// remove network
	// docker network rm <network>
	err = client.NetworkRemove(
		context.Background(),
		fh.thisNetwork,
	)
	if err != nil {
		return err
	}

	log.Println("removed network", fh.thisNetwork)

	// remove image
	// docker rmi <image>
	_, err = client.ImageRemove(
		context.Background(),
		fh.functionUniqueName,
		types.ImageRemoveOptions{},
	)

	if err != nil {
		return err
	}

	log.Println("removed image", fh.functionUniqueName)

	return nil
}

func (ms *ManagementService) createFunction(name string, env string, threads int, funczip []byte, subfolderPath string) (string, error) {

	// make a uuidv4 for the function
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	log.Println("creating function", name, "with uuid", uuid.String())

	// create a new function handler

	p := path.Join(TmpDir, uuid.String())

	err = os.MkdirAll(p, 0777)

	if err != nil {
		return "", err
	}

	log.Println("created folder", p)

	// write zip to file
	zipPath := path.Join(TmpDir, uuid.String()+".zip")
	err = os.WriteFile(zipPath, funczip, 0777)

	if err != nil {
		return "", err
	}

	err = unzip(zipPath, p)

	if err != nil {
		return "", err
	}

	if subfolderPath != "" {
		p = path.Join(p, subfolderPath)
	}

	// we know this function already, destroy its current handler
	if _, ok := ms.functionHandlers[name]; ok {
		err = ms.functionHandlers[name].destroy()
		if err != nil {
			return "", err
		}
	}

	// create new function handler
	ms.functionHandlersMutex.Lock()
	defer ms.functionHandlersMutex.Unlock()

	fh, err := create(name, env, threads, p, ms.id)

	if err != nil {
		return "", err
	}

	ms.functionHandlers[name] = fh

	err = ms.functionHandlers[name].start()

	if err != nil {
		return "", err
	}

	// tell rproxy about the new function
	// curl -X POST http://localhost:80/add -d '{"name": "<name>", "ips": ["<ip1>", "<ip2>"]}'
	d := struct {
		FunctionName string   `json:"name"`
		FunctionIPs  []string `json:"ips"`
	}{
		FunctionName: fh.functionName,
		FunctionIPs:  fh.thisHandlerIPs,
	}

	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}

	log.Println("telling rproxy about new function", fh.functionName, "with ips", fh.thisHandlerIPs, ":", d)

	resp, err := http.Post(fmt.Sprintf("http://%s:%d", ms.rproxyListenAddress, ms.rproxyConfigPort), "application/json", bytes.NewBuffer(b))
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rproxy returned status code %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Println("rproxy response:", string(r))

	// remove folder
	err = os.RemoveAll(p)
	if err != nil {
		return "", err
	}

	err = os.Remove(zipPath)
	if err != nil {
		return "", err
	}

	log.Println("removed folder", p)
	log.Println("removed zip", zipPath)

	return fh.functionName, nil
}

func (ms *ManagementService) uploadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// parse request
	d := struct {
		FunctionName    string `json:"name"`
		FunctionEnv     string `json:"env"`
		FunctionThreads int    `json:"threads"`
		FunctionZip     string `json:"zip"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	log.Println("got request to upload function: Name", d.FunctionName, "Env", d.FunctionEnv, "Threads", d.FunctionThreads, "Bytes", len(d.FunctionZip))

	// b64 decode zip
	zip, err := base64.StdEncoding.DecodeString(d.FunctionZip)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	// create function handler
	n, err := ms.createFunction(d.FunctionName, d.FunctionEnv, d.FunctionThreads, zip, "")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// return success
	w.WriteHeader(http.StatusOK)
	for prot, port := range ms.rproxyPort {
		fmt.Fprintf(w, "%s://%s:%d/%s\n", prot, ms.rproxyListenAddress, port, n)
	}

}

func (ms *ManagementService) delete(name string) error {

	fh, ok := ms.functionHandlers[name]
	if !ok {
		return fmt.Errorf("function %s not found", name)
	}

	log.Println("destroying function", fh.functionName)

	err := fh.destroy()
	if err != nil {
		return err
	}

	// tell rproxy about the delete function
	// curl -X POST http://localhost:80 -d '{"name": "<name>"}'
	d := struct {
		FunctionName string `json:"name"`
	}{
		FunctionName: fh.functionName,
	}

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	log.Println("telling rproxy about deleted function", fh.functionName)

	resp, err := http.Post(fmt.Sprintf("http://%s:%d", ms.rproxyListenAddress, ms.rproxyConfigPort), "application/json", bytes.NewBuffer(b))

	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rproxy returned status code %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Println("rproxy response:", string(r))

	delete(ms.functionHandlers, name)

	return nil
}

func (ms *ManagementService) deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// parse request
	d := struct {
		FunctionName string `json:"name"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	log.Println("got request to delete function:", d.FunctionName)

	// delete function
	err = ms.delete(d.FunctionName)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// return success
	w.WriteHeader(http.StatusOK)
}

func (ms *ManagementService) listHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (ms *ManagementService) wipe() error {
	for _, fh := range ms.functionHandlers {
		log.Println("destroying function", fh.functionName)

		err := fh.destroy()
		if err != nil {
			return err
		}

		log.Println("removed function", fh.functionName)
	}

	return nil
}

func (ms *ManagementService) wipeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := ms.wipe()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ms *ManagementService) logs() (string, error) {

	var logs string

	for _, fh := range ms.functionHandlers {
		l, err := ms.logsFunction(fh.functionName)
		if err != nil {
			return "", err
		}

		logs += l
		logs += "\n"
	}

	return logs, nil
}

func (ms *ManagementService) logsFunction(name string) (string, error) {

	fh, ok := ms.functionHandlers[name]
	if !ok {
		return "", fmt.Errorf("function %s not found", name)
	}

	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return "", err
	}

	// get container logs
	// docker logs <container>
	var logs string
	for _, container := range fh.thisContainers {
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

func (ms *ManagementService) logsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// parse request
	var logs string
	name := r.URL.Query().Get("name")

	if name == "" {
		l, err := ms.logs()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		logs = l
	}

	if name != "" {
		l, err := ms.logsFunction(name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		logs = l
	}

	// return success
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, logs)
}

func (ms *ManagementService) urlUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// parse request
	d := struct {
		FunctionName    string `json:"name"`
		FunctionEnv     string `json:"env"`
		FunctionThreads int    `json:"threads"`
		FunctionURL     string `json:"url"`
		SubFolder       string `json:"subfolder_path"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	log.Println("got request to upload function:", d)

	// download url
	resp, err := http.Get(d.FunctionURL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	zip, err := io.ReadAll(resp.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	// create function handler
	n, err := ms.createFunction(d.FunctionName, d.FunctionEnv, d.FunctionThreads, zip, d.SubFolder)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// return success
	w.WriteHeader(http.StatusOK)
	for prot, port := range ms.rproxyPort {
		fmt.Fprintf(w, "%s://%s:%d/%s\n", prot, ms.rproxyListenAddress, port, n)
	}
}
