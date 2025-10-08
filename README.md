# Go DNS Forwarder

A small, educational DNS forwarder/proxy written in Go. It listens on a local UDP socket, parses incoming DNS queries at the packet level, forwards each question to an upstream resolver and merges the responses into a single DNS reply.

This project is intentionally minimal and focuses on showing how DNS packets can be parsed and composed by hand (headers, questions and answers) and how to perform UDP-based forwarding.

---

## Features

* UDP-based DNS forwarder
* Manual DNS packet parsing & building (headers, question parsing, name compression handling)
* Forwards each question individually to an upstream resolver and merges answers
* Basic handling of flags, opcodes and RCODEs
* Timeouts & simple error handling (sends SERVFAIL on forward errors)

---

## Quick start

### Requirements

* Go 1.20+ (or recent Go toolchain)

### Build

```bash
# from project root
go build -o dns-forwarder
```

### Run

The server requires an upstream resolver address to be provided with the `--resolver` flag in the form `IP:PORT` (for example: `1.1.1.1:53` or `8.8.8.8:53`).

```bash
# run with Cloudflare
./dns-forwarder --resolver 1.1.1.1:53

# or run directly with go
go run . --resolver 8.8.8.8:53
```

By default the server listens on `127.0.0.1:2053` (non-privileged port). You can change that in the code by modifying the `runServer` call in `main`.

---

## Usage / Testing

From another shell you can query the server using `dig`:

```bash
# query for A record for example.com
dig @127.0.0.1 -p 2053 example.com A

# query with multiple record types (client may or may not send multiple questions):
# dig supports single-question queries; use other tooling for multi-question packets
```

If the forwarder cannot reach the upstream resolver or parsing fails, it will reply with a `SERVFAIL` response.

---

## Design notes & limitations

* This forwarder **forwards each question independently** to the upstream resolver and concatenates answers into a single reply. This works for the common case where queries contain a single question.
* The code has basic safeguards for malformed packets (length checks, pointer/pointer-loop detection in `ParseName`) but is not hardened for production.
* There is no caching layer. Every incoming query is forwarded to the resolver.
* No rate limiting or query filtering is implemented. Because of this, be cautious if you expose the server to an untrusted network — it could be used to amplify traffic.
* Name compression in answers is forwarded *as received* from the upstream resolver (the forwarder does not re-compress or rewrite pointers in answers). This approach is simple but may cause larger packets in some cases.
* Uses fixed read/write timeouts when talking to the upstream resolver (`2s` write, `3s` read) — these values can be tuned.

---

## Testing & debugging tips

* Use `dig` with `+norecurse` or `+tcp` flags to experiment with upstream behavior.
* To inspect raw packets, use `tcpdump` or `wireshark` while sending test queries to `127.0.0.1:2053`.
* Add logging around `forwardQuestion` and `handleRequest` to inspect question parsing and returned answer bytes.

---

## TODO / Ideas

* Add caching (LRU) for recent answers
* Support TCP transport for DNS (for large responses)
* Support configuration for listen address via CLI flag
* Add concurrency limits and statistics
* Implement better error classification and retries
* Add unit tests for packet parsing and header manipulation

* create minimal unit tests for `ParseName` / `ParseQuestion` / header helpers

Tell me which of the above you'd like and I will add it.
