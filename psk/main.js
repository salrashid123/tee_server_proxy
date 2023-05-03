const https = require("https")
const crypto = require('crypto')
const express = require("express")

const gcpMetadata = require('gcp-metadata');

// Pretend the following use the TEE's Attestation Tokens to recall alice and bob's partial keys
const {SecretManagerServiceClient} = require('@google-cloud/secret-manager').v1;
const port = 8081;
// const alice = '2c6f63f8c0f53a565db041b91c0a95add8913fc102670589db3982228dbfed90';
// const bob = 'b15244faf36e5e4b178d3891701d245f2e45e881d913b6c35c0ea0ac14224cc2';               

async function main() {

// set this to whichever projectID where the secrets are at
// for convenience, i'm setting all the secrets in one project 
// taken from the gce metadata server.  Realistically, these will be different projects for each collaborator

const project_id = 'YOUR_PROJECT_ID';

const isAvailable = await gcpMetadata.isAvailable();
if (isAvailable) {
  const projectMetadata = await gcpMetadata.project();
  project_id = projectMetadata;
}

const client = new SecretManagerServiceClient();

const [alice_version] = await client.accessSecretVersion({
  name: 'projects/' + project_id + '/secrets/alice_psk/versions/latest'
});
const alice = alice_version.payload.data.toString();


const [bob_version] = await client.accessSecretVersion({
  name: 'projects/' + project_id + '/secrets/bob_psk/versions/latest'
});
const bob = bob_version.payload.data.toString();

const key = crypto.createHash('sha256').update(alice+bob).digest('hex');
console.log(key);


// now use the combined keys to create TLS-PSK enabled "backend"
const USERS = {
  Client1: Buffer.from(key, 'hex'),
};

const options = {
   pskCallback(socket, id) {
    console.log(id);
    if (id in USERS) {
        return { psk: USERS[id] };
    }
   }
}

const app = express();

app.get('/', function (req, res) {
  console.log('connected')
  res.writeHead(200);
  res.end(`ok\n`);
})
console.log("starting server");
https.createServer(options, app).listen(port);

}

main().catch(console.error);