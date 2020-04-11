# capturereq

Capture HTTP/S requests locally, dump request to `STDOUT`, proxy to an upstream defined in a separate `hosts` file, and dump response to `STDOUT`.

Similar to Wireshark's SSL decryption function but with added proxy / request / response modification options.

## Usage

- Create a `hosts` file defining the ip / hosts for which you want to proxy requests.

- Update your system's `/etc/hosts` file to point these same domains to `127.0.0.1` (or wherever you have `capturereq` running).

- Update `.env` to reflect the path to your `capturereq` hosts file.

Now, your system's `/etc/hosts` file will redirect requests to these hosts to `localhost`.

`capturereq` will intercept the request, dump it to `STDOUT`, and then reference the locally configured `hosts` file to proxy the request to the upstream, then dump the response to `STDOUT` while responding to the initial request.
