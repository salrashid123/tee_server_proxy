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

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var (
	defaultProjectID = "YOUR_PROJECT_ID"
)

func main() {

	ctx := context.Background()

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

	// now that we have the keypair written to a file, launch envoy with a configuration
	// that will use those keys

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		cmd := exec.Command("/envoy", "-c", "/tls_proxy.yaml")
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

	go func() {
		defer wg.Done()
		cmd := exec.Command("/server")
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
