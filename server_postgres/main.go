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
	err = ioutil.WriteFile("/config/ca.pem", result.Payload.Data, 0644)
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
	err = ioutil.WriteFile("/config/server.crt", result.Payload.Data, 0644)
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
	err = ioutil.WriteFile("/config/server.key", result.Payload.Data, 0644)
	if err != nil {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	// ***************************************************************************************************

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		cmd := exec.Command("/usr/local/bin/docker-entrypoint.sh", "postgres", "--port=5432", "--ssl=on", "--ssl_cert_file=/config/server.crt", "--ssl_key_file=/config/server.key", "--ssl_ca_file=/config/ca.pem", "--hba_file=/config/pg_hba.conf")

		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "POSTGRES_PASSWORD=mysecretpassword")
		// postgres:x:999:999::/var/lib/postgresql:/bin/bash
		// cmd.SysProcAttr = &syscall.SysProcAttr{}
		// cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 999, Gid: 999}

		var stdBuffer bytes.Buffer
		mw := io.MultiWriter(os.Stdout, &stdBuffer)

		cmd.Stdout = mw
		cmd.Stderr = mw

		if err := cmd.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println(stdBuffer.String())
		err := cmd.Wait()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}()

	wg.Wait()
	fmt.Println("Process completed.")

}
