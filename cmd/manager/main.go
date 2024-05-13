package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"strings"

	"github.com/OpenFogStack/tinyFaaS/pkg/docker"
	"github.com/OpenFogStack/tinyFaaS/pkg/manager"
	"github.com/google/uuid"
)

const (
	ConfigPort          = 8080
	RProxyConfigPort    = 8081
	RProxyListenAddress = ""
)

type server struct {
	ms *manager.ManagementService
}

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("manager: ")

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

	// setting backend to docker
	id := uuid.New().String()

	// find backend
	backend, ok := os.LookupEnv("TF_BACKEND")

	if !ok {
		backend = "docker"
		log.Println("using default backend docker")
	}

	var tfBackend manager.Backend
	switch backend {
	case "docker":
		log.Println("using docker backend")
		tfBackend = docker.New(id)
	default:
		log.Fatalf("invalid backend %s", backend)
	}

	ms := manager.New(
		id,
		RProxyListenAddress,
		ports,
		RProxyConfigPort,
		tfBackend,
	)

	rproxyArgs := []string{fmt.Sprintf("%s:%d", RProxyListenAddress, RProxyConfigPort)}

	for prot, port := range ports {
		rproxyArgs = append(rproxyArgs, fmt.Sprintf("%s:%s:%d", prot, RProxyListenAddress, port))
	}

	log.Println("rproxy args:", rproxyArgs)

	// unpack the rproxy binary in a temporary directory
	rProxyDir := path.Join(os.TempDir(), id)

	err := os.MkdirAll(rProxyDir, 0755)

	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(path.Join(rProxyDir, "rproxy.bin"), RProxyBin, 0755)

	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(rProxyDir)

	c := exec.Command(path.Join(rProxyDir, "rproxy.bin"), rproxyArgs...)

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

	rproxy := c.Process

	log.Println("started rproxy")

	s := &server{
		ms: ms,
	}

	// create handlers
	r := http.NewServeMux()
	r.HandleFunc("/upload", s.uploadHandler)
	r.HandleFunc("/delete", s.deleteHandler)
	r.HandleFunc("/list", s.listHandler)
	r.HandleFunc("/wipe", s.wipeHandler)
	r.HandleFunc("/logs", s.logsHandler)
	r.HandleFunc("/uploadURL", s.urlUploadHandler)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig

		log.Println("received interrupt")
		log.Println("shutting down")

		// stop rproxy
		log.Println("stopping rproxy")
		err := rproxy.Kill()

		if err != nil {
			log.Println(err)
		}

		// stop handlers
		log.Println("stopping management service")
		err = ms.Stop()

		if err != nil {
			log.Println(err)
		}

		os.Exit(0)
	}()

	// start server
	log.Println("starting HTTP server")
	addr := fmt.Sprintf(":%d", ConfigPort)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *server) uploadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// parse request
	d := struct {
		FunctionName    string   `json:"name"`
		FunctionEnv     string   `json:"env"`
		FunctionThreads int      `json:"threads"`
		FunctionZip     string   `json:"zip"`
		FunctionEnvs    []string `json:"envs"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	log.Println("got request to upload function: Name", d.FunctionName, "Env", d.FunctionEnv, "Threads", d.FunctionThreads, "Bytes", len(d.FunctionZip), "Envs", d.FunctionEnvs)

	envs := make(map[string]string)
	for _, e := range d.FunctionEnvs {
		k, v, ok := strings.Cut(e, "=")

		if !ok {
			log.Println("invalid env:", e)
			continue
		}

		envs[k] = v
	}

	res, err := s.ms.Upload(d.FunctionName, d.FunctionEnv, d.FunctionThreads, d.FunctionZip, envs)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// return success
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, res)

}

func (s *server) deleteHandler(w http.ResponseWriter, r *http.Request) {
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
	err = s.ms.Delete(d.FunctionName)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// return success
	w.WriteHeader(http.StatusOK)
}

func (s *server) listHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	l := s.ms.List()

	// return success
	w.WriteHeader(http.StatusOK)
	for _, f := range l {
		fmt.Fprintf(w, "%s\n", f)
	}
}

func (s *server) wipeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := s.ms.Wipe()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *server) logsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// parse request
	var logs io.Reader
	name := r.URL.Query().Get("name")

	if name == "" {
		l, err := s.ms.Logs()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		logs = l
	}

	if name != "" {
		l, err := s.ms.LogsFunction(name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		logs = l
	}

	// return success
	w.WriteHeader(http.StatusOK)
	// w.Header().Set("Content-Type", "text/plain")
	_, err := io.Copy(w, logs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func (s *server) urlUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// parse request
	d := struct {
		FunctionName    string   `json:"name"`
		FunctionEnv     string   `json:"env"`
		FunctionThreads int      `json:"threads"`
		FunctionURL     string   `json:"url"`
		FunctionEnvs    []string `json:"envs"`
		SubFolder       string   `json:"subfolder_path"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	log.Println("got request to upload function:", d)

	envs := make(map[string]string)
	for _, e := range d.FunctionEnvs {
		k, v, ok := strings.Cut(e, "=")

		if !ok {
			log.Println("invalid env:", e)
			continue
		}

		envs[k] = v
	}

	res, err := s.ms.UrlUpload(d.FunctionName, d.FunctionEnv, d.FunctionThreads, d.FunctionURL, d.SubFolder, envs)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// return success
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, res)
}
