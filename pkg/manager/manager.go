package manager

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/google/uuid"
	"github.com/pfandzelter/tinyFaaS/pkg/docker"
	"github.com/pfandzelter/tinyFaaS/pkg/util"
)

const (
	TmpDir = "./tmp"
)

type ManagementService struct {
	id                    string
	createHandler         func(name string, env string, threads int, filedir string, tinyFaaSID string) (Handler, error)
	functionHandlers      map[string]Handler
	functionHandlersMutex sync.Mutex
	rproxyListenAddress   string
	rproxyPort            map[string]int
	rproxyConfigPort      int
}

type Handler interface {
	IPs() []string
	Start() error
	Destroy() error
	Logs() (string, error)
}

func New(id string, rproxyListenAddress string, rproxyPort map[string]int, rproxyConfigPort int, tfBackend string) *ManagementService {

	ms := &ManagementService{
		id:                  id,
		functionHandlers:    make(map[string]Handler),
		rproxyListenAddress: rproxyListenAddress,
		rproxyPort:          rproxyPort,
		rproxyConfigPort:    rproxyConfigPort,
	}

	if tfBackend == "docker" {
		ms.createHandler = func(name string, env string, threads int, filedir string, tinyFaaSID string) (Handler, error) {
			return docker.Create(name, env, threads, filedir, tinyFaaSID)
		}
	} else {
		log.Fatal("invalid backend", tfBackend)
	}

	return ms
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

	err = util.Unzip(zipPath, p)

	if err != nil {
		return "", err
	}

	if subfolderPath != "" {
		p = path.Join(p, subfolderPath)
	}

	// we know this function already, destroy its current handler
	if _, ok := ms.functionHandlers[name]; ok {
		err = ms.functionHandlers[name].Destroy()
		if err != nil {
			return "", err
		}
	}

	// create new function handler
	ms.functionHandlersMutex.Lock()
	defer ms.functionHandlersMutex.Unlock()

	fh, err := ms.createHandler(name, env, threads, p, ms.id)

	if err != nil {
		return "", err
	}

	ms.functionHandlers[name] = fh

	err = ms.functionHandlers[name].Start()

	if err != nil {
		return "", err
	}

	// tell rproxy about the new function
	// curl -X POST http://localhost:80/add -d '{"name": "<name>", "ips": ["<ip1>", "<ip2>"]}'
	d := struct {
		FunctionName string   `json:"name"`
		FunctionIPs  []string `json:"ips"`
	}{
		FunctionName: name,
		FunctionIPs:  fh.IPs(),
	}

	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}

	log.Println("telling rproxy about new function", name, "with ips", fh.IPs(), ":", d)

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

	return name, nil
}

func (ms *ManagementService) Logs() (string, error) {

	var logs string

	for name := range ms.functionHandlers {
		l, err := ms.LogsFunction(name)
		if err != nil {
			return "", err
		}

		logs += l
		logs += "\n"
	}

	return logs, nil
}

func (ms *ManagementService) LogsFunction(name string) (string, error) {

	fh, ok := ms.functionHandlers[name]
	if !ok {
		return "", fmt.Errorf("function %s not found", name)
	}

	return fh.Logs()
}

func (ms *ManagementService) Wipe() error {
	for name := range ms.functionHandlers {
		log.Println("destroying function", name)
		ms.Delete(name)
	}

	return nil
}
func (ms *ManagementService) Delete(name string) error {

	fh, ok := ms.functionHandlers[name]
	if !ok {
		return fmt.Errorf("function %s not found", name)
	}

	log.Println("destroying function", name)

	ms.functionHandlersMutex.Lock()
	defer ms.functionHandlersMutex.Unlock()

	err := fh.Destroy()
	if err != nil {
		return err
	}

	// tell rproxy about the delete function
	// curl -X POST http://localhost:80 -d '{"name": "<name>"}'
	d := struct {
		FunctionName string `json:"name"`
	}{
		FunctionName: name,
	}

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	log.Println("telling rproxy about deleted function", name)

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

func (ms *ManagementService) Upload(name string, env string, threads int, zipped string) (string, error) {

	// b64 decode zip
	zip, err := base64.StdEncoding.DecodeString(zipped)
	if err != nil {
		// w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return "", err
	}

	// create function handler
	n, err := ms.createFunction(name, env, threads, zip, "")

	if err != nil {
		// w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return "", err
	}

	// return success
	// w.WriteHeader(http.StatusOK)
	r := ""
	for prot, port := range ms.rproxyPort {
		r += fmt.Sprintf("%s://%s:%d/%s\n", prot, ms.rproxyListenAddress, port, n)
	}

	return r, nil
}

func (ms *ManagementService) UrlUpload(name string, env string, threads int, funcurl string, subfolder string) (string, error) {

	// download url
	resp, err := http.Get(funcurl)
	if err != nil {
		// w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return "", err
	}

	// reading body to memory
	// not the smartest thing
	zip, err := io.ReadAll(resp.Body)

	if err != nil {
		// w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return "", err
	}

	// create function handler
	n, err := ms.createFunction(name, env, threads, zip, subfolder)

	if err != nil {
		// w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return "", err
	}

	// return success
	// w.WriteHeader(http.StatusOK)
	r := ""
	for prot, port := range ms.rproxyPort {
		r += fmt.Sprintf("%s://%s:%d/%s\n", prot, ms.rproxyListenAddress, port, n)
	}

	return r, nil
}
