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

	"github.com/OpenFogStack/tinyFaaS/pkg/util"
	"github.com/google/uuid"
)

const (
	TmpDir = "./tmp"
)

type ManagementService struct {
	id                    string
	backend               Backend
	functionHandlers      map[string]Handler
	functionHandlersMutex sync.Mutex
	rproxyListenAddress   string
	rproxyPort            map[string]int
	rproxyConfigPort      int
}

type Backend interface {
	Create(name string, env string, threads int, filedir string, envs map[string]string) (Handler, error)
	Stop() error
}

type Handler interface {
	IPs() []string
	Start() error
	Destroy() error
	Logs() (io.Reader, error)
}

func New(id string, rproxyListenAddress string, rproxyPort map[string]int, rproxyConfigPort int, tfBackend Backend) *ManagementService {

	ms := &ManagementService{
		id:                  id,
		backend:             tfBackend,
		functionHandlers:    make(map[string]Handler),
		rproxyListenAddress: rproxyListenAddress,
		rproxyPort:          rproxyPort,
		rproxyConfigPort:    rproxyConfigPort,
	}

	return ms
}

func (ms *ManagementService) createFunction(name string, env string, threads int, funczip []byte, subfolderPath string, envs map[string]string) (string, error) {

	// only allow alphanumeric characters
	if !util.IsAlphaNumeric(name) {
		return "", fmt.Errorf("function name %s contains non-alphanumeric characters", name)
	}

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

	defer func() {
		// remove folder
		err = os.RemoveAll(p)
		if err != nil {
			log.Println("error removing folder", p, err)
		}

		err = os.Remove(zipPath)
		if err != nil {
			log.Println("error removing zip", zipPath, err)
		}

		log.Println("removed folder", p)
		log.Println("removed zip", zipPath)
	}()

	if subfolderPath != "" {
		p = path.Join(p, subfolderPath)
	}

	// if function already exists, keep it while deploying the new version
	var oldHandler Handler
	if existingHandler, ok := ms.functionHandlers[name]; ok {
		oldHandler = existingHandler
	}

	// create new function handler
	ms.functionHandlersMutex.Lock()
	defer ms.functionHandlersMutex.Unlock()

	fh, err := ms.backend.Create(name, env, threads, p, envs)

	if err != nil {
		return "", err
	}

	ms.functionHandlers[name] = fh

	err = ms.functionHandlers[name].Start()

	if err != nil {
		// container did not start properly...
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
		log.Println("error telling rproxy about new function", name, err)
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

	// destroy the old handler if it exists
	if oldHandler != nil {
		err = oldHandler.Destroy()
		if err != nil {
			return "", err
		}
	}

	return name, nil
}

func (ms *ManagementService) Logs() (io.Reader, error) {

	var logs bytes.Buffer

	for name := range ms.functionHandlers {
		l, err := ms.LogsFunction(name)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(&logs, l)
		if err != nil {
			return nil, err
		}

		logs.WriteString("\n")
	}

	return &logs, nil
}

func (ms *ManagementService) LogsFunction(name string) (io.Reader, error) {

	fh, ok := ms.functionHandlers[name]
	if !ok {
		return nil, fmt.Errorf("function %s not found", name)
	}

	return fh.Logs()
}

func (ms *ManagementService) List() []string {
	list := make([]string, 0, len(ms.functionHandlers))
	for name := range ms.functionHandlers {
		list = append(list, name)
	}

	return list
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

func (ms *ManagementService) Upload(name string, env string, threads int, zipped string, envs map[string]string) (string, error) {

	// b64 decode zip
	zip, err := base64.StdEncoding.DecodeString(zipped)
	if err != nil {
		// w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return "", err
	}

	// create function handler
	n, err := ms.createFunction(name, env, threads, zip, "", envs)

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

func (ms *ManagementService) UrlUpload(name string, env string, threads int, funcurl string, subfolder string, envs map[string]string) (string, error) {

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
	n, err := ms.createFunction(name, env, threads, zip, subfolder, envs)

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

func (ms *ManagementService) Stop() error {
	err := ms.Wipe()
	if err != nil {
		return err
	}

	return ms.backend.Stop()
}
