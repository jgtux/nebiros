Traffic obfuscation over QUIC or MASQUE.

Modes

- `quic`: raw QUIC datagrams using the existing transport.
- `masque`: HTTP/3 `CONNECT-UDP` proxying via MASQUE.

Examples

Server, raw QUIC:

```sh
./server -mode quic -listen :4433 -target 127.0.0.1:51820
```

Client, raw QUIC:

```sh
./client -mode quic -listen 127.0.0.1:51821 -server 127.0.0.1:4433 -server-name localhost -insecure
```

Server, MASQUE / HTTP/3:

```sh
./server -mode masque -listen :4433 -target 127.0.0.1:51820 -proxy-template 'https://localhost:4433/masque?h={target_host}&p={target_port}'
```

Client, MASQUE / HTTP/3:

```sh
./client -mode masque -listen 127.0.0.1:51821 -target 127.0.0.1:51820 -proxy-template 'https://localhost:4433/masque?h={target_host}&p={target_port}' -server-name localhost -insecure
```

Notes

- In `masque` mode, client and server must use the same URI template.
- The MASQUE server currently only allows the configured `-target`.
- Both modes keep the local UDP bridge behavior intact.
