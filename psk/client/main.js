const tls = require('tls');
const crypto = require('crypto')
const host = "localhost";
const port = 8081;


const alice = '2c6f63f8c0f53a565db041b91c0a95add8913fc102670589db3982228dbfed90';
const bob = 'b15244faf36e5e4b178d3891701d245f2e45e881d913b6c35c0ea0ac14224cc2';               
const key = crypto.createHash('sha256').update(alice+bob).digest('hex');

const USERS = {
  Client1: Buffer.from(key, 'hex'),
};
const options = {
    pskCallback(socket, id) {
        return { psk: USERS.Client1, identity: 'Client1' };
    }
}
async function main() {
var client = tls.connect(port,host, options, function () {
    client.write("GET / HTTP/1.0\n\n");
});

client.on("data", function (data) {
    console.log('Received: %s ',
        data.toString().replace(/(\n)/gm, ""));
    client.end();
});

client.on('close', function () {
    console.log("Connection closed");
});

client.on('error', function (error) {
    console.error(error);
    client.destroy();
});

//*********************** */

const https = require('https');
https.get('https://' + host + ':' + port + '/',options, (res) => {
    console.log('statusCode:', res.statusCode);
    res.on('data', (d) => {
        process.stdout.write(d);
    });
}).on('error', (e) => {
    console.error(e);
});

}

main().catch(console.error);