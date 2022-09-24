# slink

`slink` is a link shortening app where the short links are generated
privately/internally only.

It’s designed for secure, observable, and scalable production use.

## Features

- [Nano ID](https://github.com/ai/nanoid) based generated short link IDs
  - URL-safe default character set
  - custom (ASCII) character set
  - configurable length IDs, can be changed over time
  - avoid generated IDs matching a list of words (denylist)
  - supply your own denylist or use the default
- [Amazon DynamoDB](https://aws.amazon.com/dynamodb/) storage backend
  - create your own table (must conform to certain structure, recommended)
  - auto create table if missing (requires permission, mainly for development)
  - works with `amazon/dynamodb-local` for local development and testing
  - single table pattern based on best practices recommended by
    [The DynamoDB Book](https://dynamodbbook.com)
- Expiring links
  - an optional `ExpiresAt` can be supplied when creating ShortLink
  - the public server will only redirect when the ShortLink has no expiry or is not yet expired
- Fallback redirect URL for missing or expired links
  - or respond 404 when the fallback URL is not specified
- LRU cache for public lookups/redirects (in-memory, per-process only)
- Built-in Prometheus (operational) metrics and pprof
  - running on a separate debug server in the same processs
  - the debug server is optional, it's off by default
- Docker image/container based deployment (including Kubernetes)
  - small image size (less than 10MB)
  - separate binaries for public facing and admin servers
  - stateless public and admin apps, easy to horizontally scale
  - configurable runtime behaviour via flags and/or a JSON configuration file
- Use your own domain
- Use `slink` as a library, extend it, build your own
  - supply your own short ID generator
  - supply your own storage backend
  - supply your own tracking mechanism
- Opinionated reasonable defaults for production

## What's missing?

I’m planning to add these (eventually):

- Kubernetes deployment:
  - documentation
- tests
- preventing too many redirects to self
- more admin APIs:
  - update short link target URL (correct mistake on already published short link, swap target after embargo date, etc)
  - expire short links by ID (due to mistake, abuse, disappearing target, etc)
  - expire short links by URL
- normalising Link URLs before generating a ShortLink for it
  - lower/upper casing
  - query params ordering

I have no plans to add these:

- built-in analytics (beyond tracking)
  - it should live outside of `slink`
  - there's already a way to track
    - can implement your own if you need anything other than SNS
- web-based admin UI
  - it should live in a separate project
- CLI binary
  - use `curl`, etc, to interact with the HTTP API
- pluggable LRU cache for public lookups/redirects (e.g. Redis)
  - consider [DynamoDB DAX](https://aws.amazon.com/dynamodb/dax/) so no
    change is needed in `slink`
    - with DAX, the LRU Cache in `slink` is unnecessary but it should be
      relatively harmless
  - I’m somewhat open to making changes to make it possible to supply your own
    LRU Cache, if you can convince me with a solid use case 🙂
- built-in option to auto delete expired `ShortLink` using DynamoDB TTL
  - storage is relatively cheap
  - create your own table and configure `ExpiresAt` to be TTL if you need it
- ability to create short links in the public facing app (like bit.ly for example,
  allowing anyone to create short links)
  - the security requirements is quite different
  - if you really want to do it, you can create your own public server using `slink`
    components
- multi-tenancy
- fine-grained permissions
  - any authenticated client that can hit the Admin API can do everything
  - do it elsewhere (proxy or client)

## The public and admin servers

TODO: Why?

TODO: What?

## Configuration

`slink` uses [`ff`](https://github.com/peterbourgon/ff) with JSON config file option for the configuration so that you
can specify the configuration either via CLI flags, or via a JSON file, or both (the CLI flags take precedence).


### `slink-public-server`

See the `FlagSet` in [cmd/slink-admin-server/main.go](./cmd/slink-admin-server/main.go) for all available options.

Or, you can also do `go run cmd/slink-public-server -help`.

### `slink-admin-server`

See the FlagSet in [cmd/slink-public-server/main.go](./cmd/slink-public-server/main.go) for all available options.

Or, you can also do `go run cmd/slink-admin-server -help`.

## Kubernetes Deployment

TODO

## Docker

TODO: link to Docker hub.

TODO: update instructions below to use the public docker image.

Run `slink-admin-server` locally against dynamodb-local (assuming dynamodb-local
is running at port 8000 on the Docker host):

```sh
docker run --rm -it -p 9090:9090 slink slink-admin-server \
  -dynamodb-endpoint http://host.docker.internal:8000 \
  -aws-access-key-id slink
```

Run `slink-public-server` locally against dynamodb-local (assuming dynamodb-local
is running at port 8000 on the Docker host):

```sh
docker run --rm -it -p 8080:8080 slink slink-public-server \
  -dynamodb-endpoint http://host.docker.internal:8000 \
  -aws-access-key-id slink
```

## Running dynamodb-local

The easiest way to run `dynamodb-local` is probably by using their public docker
image, run as a daemon, with a shared local data directory on the host for
persistence. This single container can be used to develop multiple applications
at the same time (dynamodb-local uses `AWS_ACCESS_KEY_ID` to separate the data
into different namespaces), so you should only need one per development machine.

```
docker run -d \
  --name ddblocal
  -p 8000:8000 \
  -v "/path/to/dynamodb-local:/home/dynamodblocal/data"
  amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb -dbPath ./data

```

See https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/DynamoDBLocal.DownloadingAndRunning.html
for more information and options.
