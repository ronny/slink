# slink

`slink` is a link or URL shortening app.

The only supported storage backend for production is Amazon DynamoDB.

## Docker

Run `slink-admin-server` locally against dynamodb-local (assuming dynamodb-local
is running at port 8000 on the host):

```
docker run --rm -it -p 9090:9090 slink slink-admin-server \
  -dynamodb-endpoint http://host.docker.internal:8000 \
  -aws-access-key-id slink
```

Run `slink-public-server` locally against dynamodb-local (assuming dynamodb-local
is running at port 8000 on the host):

```
docker run --rm -it -p 8080:8080 slink slink-admin-server \
  -dynamodb-endpoint http://host.docker.internal:8000 \
  -aws-access-key-id slink
```
