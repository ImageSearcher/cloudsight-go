cloudsight-go
=============

[![GoDoc](https://godoc.org/github.com/cloudsight/cloudsight-go?status.png)](https://godoc.org/github.com/cloudsight/cloudsight-go)

A simple CloudSight API Client for Go programming language.

Status
======

This package is currently in **beta** status. It means the API may still change
in **backwards incompatible** way.

Installation
============

```
$ go get github.com/cloudsight/cloudsight-go
```

Configuration
=============

You need your API key and secret (if using OAuth1 authentication). They are
available on [CloudSight site](https://cloudsightapi.com) after you sign up and
create a project.

Usage
=====

Import the `cloudsight` package:

```go
import (
    ...

    "github.com/cloudsight/cloudsight-go"
)
```

Create a client instance using simple key-based authentication:

```go
client, err := cloudsight.NewClientSimple("your-api-key")
```

Or, using OAuth1 authentication:

```go
client, err := cloudsight.NewClientOAuth("your-api-key", "your-api-secret")
```

Send the image request using a file:

```go
f, err := os.Open("your-file.jpg")
if err != nil {
	panic(err)
}

defer f.Close()

params := cloudsight.Params{}
params.SetLocale("en")
job, err := client.ImageRequest(f, "your-file.jpg", params)
```

Or, you can send the image request using a URL:

```go
params := cloudsight.Params{}
params.SetLocale("en")
job, err := client.RemoteImageRequest("http://www.example.com/image.jpg", params)
```

Then, update the job status to see if it's already processed:

```go
err := client.UpdateJob(job)
if job != cloudsight.StatusNotCompleted {
	// Done!
}
```

It usually takes 6-12 seconds to receive a completed response. You may use
`WaitJob()` method to wait until the image is processed:

```go
err := client.WaitJob(job, 30 * time.Second)
```

Note that `Params` is just a `map[string]string` so you may choose to use more
"dynamic" approach:

```go
params := map[string]string{
	"image_request[locale]": "en",
}
job, err := client.RemoteImageRequest("http://www.example.com/image.jpg", cloudsight.Params(params))
```

Consult the [complete documentation](https://godoc.org/github.com/cloudsight/cloudsight-go "cloudsight-go documentation").
