package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"cloud.google.com/go/compute/metadata"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var ()

func main() {

	// set this to whichever projectID where the secrets are at
	// for convenience, i'm setting all the secrets in one project
	// taken from the gce metadata server.  Realistically, these will be different projects for each collaborator

	defaultProjectID := "YOUR_PROJECT_ID" // change this

	ctx := context.Background()

	if metadata.OnGCE() {
		var err error
		defaultProjectID, err = metadata.ProjectID()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	// pretend the following uses the TEE's Attestation Service to retrieve the mTLS keypair from a remote system
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer client.Close()

	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/ca/versions/latest", defaultProjectID),
	}

	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	err = ioutil.WriteFile("/ca.pem", result.Payload.Data, 0644)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	accessRequest = &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/server-cert/versions/latest", defaultProjectID),
	}

	result, err = client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	err = ioutil.WriteFile("/server.crt", result.Payload.Data, 0644)
	if err != nil {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	accessRequest = &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/server-key/versions/latest", defaultProjectID),
	}

	result, err = client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	err = ioutil.WriteFile("/server.key", result.Payload.Data, 0644)
	if err != nil {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	// ***************************************************************************************************

	// now that we have the keypair written to a file, launch redis with a configuration
	// that will use those keys

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		cmd := exec.Command("/redis-server", "--tls-port", "16379", "--tls-cert-file", "/server.crt", "--tls-key-file", "/server.key", "--tls-ca-cert-file", "/ca.pem")

		var stdBuffer bytes.Buffer
		mw := io.MultiWriter(os.Stdout, &stdBuffer)

		cmd.Stdout = mw
		cmd.Stderr = mw

		if err := cmd.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println(stdBuffer.String())
		err = cmd.Wait()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}()

	wg.Wait()
	fmt.Println("Process completed.")

}
