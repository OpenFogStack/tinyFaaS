package dockerkv

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	fred "git.tu-berlin.de/mcc-fred/fred/proto/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func createClient(host string, certFile string, keyFile string, caFiles []string) (fred.ClientClient, error) {
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)

	if err != nil {
		return nil, err
	}

	// Create a new cert pool and add our own CA certificate
	rootCAs := x509.NewCertPool()

	for _, f := range caFiles {
		loaded, err := os.ReadFile(f)

		if err != nil {
			return nil, err
		}

		rootCAs.AppendCertsFromPEM(loaded)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    rootCAs,
		RootCAs:      rootCAs,
		MinVersion:   tls.VersionTLS12,
		// TODO
		InsecureSkipVerify: true,
	}

	creds := credentials.NewTLS(config)

	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(creds))

	if err != nil {
		return nil, err
	}

	return fred.NewClientClient(conn), nil
}

func (db *DockerKVBackend) createKeygroup(keygroup string) error {
	_, err := db.fredClient.CreateKeygroup(context.Background(), &fred.CreateKeygroupRequest{
		Keygroup: keygroup,
		Mutable:  true,
		Expiry:   0,
	})

	if err != nil {
		log.Println("error creating keygroup", err)
		return fmt.Errorf("error creating keygroup: %w", err)
	}

	return nil
}

func (db *DockerKVBackend) addUserToKeygroup(keygroup string, user string) error {

	log.Printf("adding user %s to keygroup %s", user, keygroup)

	for _, perm := range []fred.UserRole{
		fred.UserRole_ReadKeygroup,
		fred.UserRole_WriteKeygroup,
		fred.UserRole_ConfigureReplica,
	} {
		_, err := db.fredClient.AddUser(context.Background(), &fred.AddUserRequest{
			Keygroup: keygroup,
			User:     user,
			Role:     perm,
		})

		if err != nil {
			log.Println("error adding user to keygroup", err)
			return fmt.Errorf("error adding user to keygroup: %w", err)
		}
	}

	return nil
}
