
# mTLS proxy containers for GCP Confidential Compute

This repo is a basic demo of a type of "systemd" for a container which seeks to simply bootstrap mTLS certificates for the 'background' application to use.

Essentially, this is either a go (or nodejs) application running in a container that on start, reads mTLS certificates from GCP [Secret Manager](https://cloud.google.com/secret-manager).

Once it read the certificates, it will then provide them to background applications to use.

Basically its a way to acquire mTLS certificates for applications that may not have the capability to specifically read them from GCP.


>> this repo is *not* supported by google

- Its not good container practice to run multiple applications in one container...anyway why do all this and not just `systemd`?

Good question!....the intent for all this is to use with GCP [Confidential Space](https://cloud.google.com/docs/security/confidential-space) instances.  These VMs just run containers with a specific profile for very secure workloads.

One characteristic is that the not even the VM operator can access the runtime (eg ssh in) and the runtime container can demonstrate that it is running a specific workload prior to receiving sensitive data or decryption keys.

Each container is treated as a `Trusted Execution Environment (TEE)` is initially startsup with no credentials (no certificates, no decryption keys).  It uses the TEE's [Attestation Service](https://cloud.google.com/docs/security/confidential-space#attestation-process) to 'prove' to a remote system the application is in a trusted environment.  Once that step is done, it will receive decryption keys or credentials.

For `TEE --> TEE` traffic, one such credential would be mTLS keypairs issued by the remote party.  This ensures each TEE is actually communicating with another authorized TEE (and vice versa).

Its fairly well know how an application can retrieve or decrypt data within a GCP TEE:

- [Constructing Trusted Execution Environment (TEE) with GCP Confidential Space](https://github.com/salrashid123/confidential_space#mtls-using-acquired-keys)
- [Multiparty Consent Based Networks (MCBN)](https://github.com/salrashid123/mcbn)

but the real issue this repo addresses is what happens if the "application" (redis, mysql, etc) has no built in mechanism to acquire mTLS keypairs thorough Confidential Space?

In this case, we needed a 'bootstrap' application which understands how to interact with GCP Confidential Space, acquire the credentials, and then furbish it to the backend application.

The bootstrap application in this case is a simple golang app that uses the attestation token to retrieve mTLS secrets from a collaborators Secret Manager.

Once it acquires the mTLS keys, it will [os/exec](https://pkg.go.dev/os/exec#Cmd.Run) the background application and await its completion.

Basically, is a container which starts a go application that gets keys, provides those keys to a background application and launches the background app.

See

- [Run multiple services in a container](https://docs.docker.com/config/containers/multi-service_container/)


- As for why not use `systemd`? 

Well, running systemd requires **a lot** of additional software which can compromise the TEE itself.  Its far better to use minimal container surface.  In most of the examples, i just "copied" the static-compiled binary over into a [distroless/base](https://github.com/GoogleContainerTools/distroless).  
 
I recognize not all background services can run on distroless images...

---


Anyway, there are four variations described here with different background applications and key requirements.

* `Redis`
  
  In ths mode the forground application acquires keys and provides those keys to redis as its own startup arguments for mTLS


* `Envoy (HTTPFilter)`

  The foreground application acquires keys and writes the keys to the filesystem.  Envoy [HTTPFilter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/http_filters) is started which uses those keys for mTLS

* `Envoy (NetworkFilter)`

  The foreground application acquires keys and writes the keys to the filesystem.  Envoy [NetworkFilter](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/network_filters/network_filters) is started which uses those keys for mTLS

* [Multiparty Consent Based Networks (MCBN)](https://github.com/salrashid123/mcbn)

  This mode describes a very rare situation where multiple collaborators each provide a partial key to use for TLS between TEEs.  In this, mTLS is not used in the traditional RSA keypair mode but the actual `TEE->TEE` traffic is only allowed if each collaborator provides their key share. 


### Setup


We will primarily use GCP Secret Manager to store the mTLS secrets so upload those

```bash
gcloud secrets create ca --replication-policy=automatic   --data-file=certs/ca.pem
gcloud secrets create server-cert --replication-policy=automatic   --data-file=certs/server.crt
gcloud secrets create server-key --replication-policy=automatic   --data-file=certs/server.key

gcloud auth application-default login
```

Then edit the following in ths repo and replace the `PROJECT_ID` value you use

```bash
# edit the following
- server_envoy_http_proxy/main.go
- server_envoy_network_proxy/main.go
- server_redis/main.go
- psk/main.js
```

---

### client -> container(envoy_redis_proxy -> redis_backend) 

For Redis, build the container and test:

```bash
cd server_redis/
docker build -t  server_redis .

docker run -ti -v "$HOME"/.config/gcloud:/root/.config/gcloud  -p 16379:16379 server_redis
```

At this point the redis container listens on `:16379` for mTLS traffic where the keys were downloaded from Secret Manger

You can verify using the test redis client:

```bash
cd client
go run main.go
```

---

### client -> container(envoy_http_proxy -> http_backend) 

For envoy HTTP backend, the envoy config simply sends a static echo response back

```bash
cd server_envoy_http_proxy/

docker build -t  server_envoy_http_proxy .
docker run -ti -v "$HOME"/.config/gcloud:/root/.config/gcloud   -p 8081:8081 server_envoy_http_proxy
```

Now that the envoy background process is running, send mTLS traffic

```bash
curl -v     --connect-to server.domain.com:443:127.0.0.1:8081 --cacert certs/ca.pem  \
    --cert certs/client.crt  \
    --key certs/client.key  https://server.domain.com/ 
```
---

### client -> container(envoy_network_proxy -> tcp_backend) 

For envoy Network backend, we will launch two processes in the background:  envoy in TCP mode which handles mTLS and a background TCP application _which just happens to be an http server_.

```bash
cd server_envoy_network_proxy/

cd backend_app/
GOOS=linux GOARCH=amd64 go build -o server
cp server ../
rm server

docker build -t  server_envoy_network_proxy .
docker run -ti -v "$HOME"/.config/gcloud:/root/.config/gcloud   -p 8081:8081 server_envoy_network_proxy
```

Now that the envoy background process is running, send mTLS traffic (again, we just happen to be running an http TCP server so curl works here)

```bash
curl -v     --connect-to server.domain.com:443:127.0.0.1:8081 --cacert certs/ca.pem  \
    --cert certs/client.crt  \
    --key certs/client.key  https://server.domain.com/ 
```


### client -> container(node_psk -> echo) 

This demonstrates the multiparty consent network described earlier.

For this, we need to seed secret manager with each collaborators partal `TLS-PSK` keys

```bash
cd psk/

echo -n "2c6f63f8c0f53a565db041b91c0a95add8913fc102670589db3982228dbfed90" > alice.psk
echo -n "b15244faf36e5e4b178d3891701d245f2e45e881d913b6c35c0ea0ac14224cc2" > bob.psk

gcloud secrets create alice_psk --replication-policy=automatic   --data-file=alice.psk
gcloud secrets create bob_psk --replication-policy=automatic   --data-file=bob.psk
```

Then build and run the node application.  The node application simulates the go application above but we are only using node as the "only application" since very few systems support defining TLS-PSK keys (see  [golang#6379](https://github.com/golang/go/issues/6379), [envoy#13237](https://github.com/envoyproxy/envoy/issues/13237)).  Ideally you could use the node app as a TCP proxy for a background application you would run.  This is not demonstrated in thsi repo

```bash
docker build -t  psk_nodejs .
docker run -ti -v "$HOME"/.config/gcloud:/root/.config/gcloud   -p 8081:8081 psk_nodejs
```

Now that the node app has all the keys, run the client

```bash
cd client/
node main.js
```
