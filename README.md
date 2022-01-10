# Leicht-Cloud

This is still pre-alpha software as of now.
The aim is to provide a Nextcloud/Owncloud like alternative with similar/more modularity.

## Development

To run this locally you'll have to install [golang](https://golang.org) and [protobuf](https://developers.google.com/protocol-buffers) locally first.
After you've done this, it just takes a few more commands to build the project and get up and running.
As the project matures over time I will provide binaries to download directly, making these steps only required if you want to work on the project itself.

```bash
./update-assets.sh
go install github.com/golang/protobuf/proto
go install github.com/golang/protobuf/protoc-gen-go
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
go generate ./...
go build ./cmd/leicht-cloud/....
# if you want to work on the html, build it with instead
# this will cause the assets to be read from the filesystem
# instead of being embedded in the binary
go build -tags=html ./cmd/leicht-cloud/...
```

### Plugins

As leicht-cloud is meant to be modular, we support plugins.
As of right now only for the underlying storage layer.
To implement this I would redirect you to the grpc interface over [here](./pkg/storage/plugin/service.proto).
If using golang directly you won't have to think about this and can use the provided wrapper, I encourage you to look at the provided plugins.

Note that these interfaces will likely still change and should not be considered stable.

### Frontend

Frontend is my absolute weak point and I could absolutely use some help here.
In case you would like to help out, feel free to reach out to me.
I would however preferably not have the entire interface depend on Javascript and ideally have most of it work without Javascript enabled at all.
