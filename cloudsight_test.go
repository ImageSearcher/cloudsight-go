package cloudsight

import (
	"fmt"
	"os"
	"time"
)

func ExampleClient_ImageRequest() {
	c, _ := NewClientSimple("api-key")
	f, _ := os.Open("some-file.jpg")
	defer f.Close()

	params := Params{}
	// Set the position to 50.0°N 19.0°E
	params.SetLatitude(50.0)
	params.SetLongitude(19.0)

	job, err := c.ImageRequest(f, "some-file.jpg", params)
	if err != nil {
		panic(err)
	}

	fmt.Println("Token:", job.Token)
}

func ExampleClient_RemoteImageRequest() {
	c, _ := NewClientSimple("api-key")
	params := Params{}
	// Set the position to 50.0°N 19.0°E
	params.SetLatitude(50.0)
	params.SetLongitude(19.0)

	job, err := c.RemoteImageRequest("http://www.example.com/some-image.jpg", params)
	if err != nil {
		panic(err)
	}

	fmt.Println("Token:", job.Token)
}

func ExampleClient_UpdateJob() {
	c, _ := NewClientSimple("api-key")
	job, _ := c.RemoteImageRequest("http://www.example.com/some-image.jpg", nil)

	time.Sleep(3 * time.Second)

	for job.Status == StatusNotCompleted {
		time.Sleep(1 * time.Second)
		c.UpdateJob(job)
	}

	fmt.Println("Status:", job.Status, job.Status.Description())
}
