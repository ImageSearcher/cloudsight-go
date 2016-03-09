// A simple CloudSight API client library. For full API documentation go to
// https://cloudsight.readme.io/v1.0/docs.
package cloudsight

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Base API URL.
const BaseURL = "https://api.cloudsightapi.com"

// Default locale that will be used when none is specified in Params.
const DefaultLocale = "en-US"

const (
	requestsURL  = BaseURL + "/image_requests"
	responsesURL = BaseURL + "/image_responses/"
)

const pollMinWait = time.Second * 4

var (
	ErrMissingKey          = errors.New("key cannot be empty")
	ErrMissingSecret       = errors.New("secret cannot be empty")
	ErrTimeout             = errors.New("poll timed out")
	ErrInvalidRepostStatus = errors.New("the job needs to have the timeout status")
)

var userAgent = []string{"cloudsight-go v1.0"}

type apiResponse struct {
	Categories []string        `json:"categories"`
	Error      json.RawMessage `json:"error"`
	Name       string          `json:"name"`
	Reason     string          `json:"reason"`
	Status     string          `json:"status"`
	TTL        float64         `json:"ttl"`
	Token      string          `json:"token"`
	URL        string          `json:"url"`
}

// Possible values for current job status.
type JobStatus string

const (
	// Recognition has not yet been completed for this image. Continue polling
	// until response has been marked completed.
	StatusNotCompleted JobStatus = "not completed"

	// Recognition has been completed. Annotation can be found in Name and
	// Categories field of Job structure.
	StatusCompleted JobStatus = "completed"

	// Token supplied on URL does not match an image.
	StatusNotFound JobStatus = "not found"

	// Image couldn't be recognized because of a specific reason. Check the
	// SkipReason field.
	StatusSkipped JobStatus = "skipped"

	// Recognition process exceeded the allowed TTL setting.
	StatusTimeout JobStatus = "timeout"
)

// Return a detailed description of the job status.
func (s JobStatus) Description() string {
	switch s {
	case StatusNotCompleted:
		return "Recognition has not yet been completed for this image. Continue polling until response has been marked completed."
	case StatusCompleted:
		return "Recognition has been completed. Annotation can be found in name element of the JSON response."
	case StatusNotFound:
		return "Token supplied on URL does not match an image."
	case StatusSkipped:
		return "Image couldn't be recognized because of a specific reason. Check the SkipReason field."
	case StatusTimeout:
		return "Recognition process exceeded the allowed TTL setting."
	default:
		return fmt.Sprintf("Unknown status: %d.", s)
	}
}

// The API may choose not to return any response for given image. SkipReason
// type includes possible reasons for such behavior.
type SkipReason string

const (
	// Offensive image content.
	ReasonOffensive SkipReason = "offensive"

	// Too blurry to identify.
	ReasonBlurry SkipReason = "blurry"

	// Too close to identify.
	ReasonClose SkipReason = "close"

	// Too dark to identify.
	ReasonDark SkipReason = "dark"

	// Too bright to identify.
	ReasonBright SkipReason = "bright"

	// Content could not be identified.
	ReasonUnsure SkipReason = "unsure"
)

// Return a detailed description of the skip reason.
func (r SkipReason) Description() string {
	switch r {
	case "":
		return "The image hasn't been skipped."
	case ReasonOffensive:
		return "Offensive image content."
	case ReasonBlurry:
		return "Too blurry to identify."
	case ReasonClose:
		return "Too close to identify."
	case ReasonDark:
		return "Too dark to identify."
	case ReasonBright:
		return "Too bright to identify."
	case ReasonUnsure:
		return "Content could not be identified."
	default:
		return fmt.Sprintf("Unknown reason: %d.", r)
	}
}

type Client struct {
	key    string
	secret string
}

// Job is a result of sending an image to CloudSight API.
type Job struct {
	// Image categories as annotated by the API.
	Categories []string

	// Image description as annotated by the API.
	Name string

	// Current job status.
	Status JobStatus

	// Time To Live.
	TTL float64

	// Token uniquely identifying the job.
	Token string

	// URL to the image as stored on CloudSight API servers.
	URL string

	// The reason for the job being skipped, if any.
	SkipReason SkipReason

	createdAt time.Time
	mu        *sync.Mutex
}

// Create a new Client instance that will authenticate using OAuth1 protocol.
//
// Error (ErrMissingKey or ErrMissingSecret) will be returned if either key or
// secret is empty.
func NewClientOAuth(key, secret string) (*Client, error) {
	if key == "" {
		return nil, ErrMissingKey
	}
	if secret == "" {
		return nil, ErrMissingSecret
	}
	return &Client{
		key:    key,
		secret: secret,
	}, nil
}

// Create a new Client instance that will authenticate using simple key-based
// method.
//
// ErrMissingKey will be returned if key is empty.
func NewClientSimple(key string) (*Client, error) {
	if key == "" {
		return nil, ErrMissingKey
	}
	return &Client{
		key: key,
	}, nil
}

func (c *Client) getAuthHeader(method, url string, params Params) (string, error) {
	if c.secret == "" {
		// Use simple authentication
		return fmt.Sprintf("CloudSight %s", c.key), nil
	} else {
		// Use OAuth1 authentication
		return oauthSign(method, url, c.key, c.secret, params)
	}
}

func (c *Client) doImageRequest(req *http.Request) (*Job, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var job apiResponse

	if err := decoder.Decode(&job); err != nil {
		return nil, err
	}

	if job.Error != nil {
		return nil, fmt.Errorf("api error: %s, status code: %d", string(job.Error), resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println("status code", resp.StatusCode)
	}

	jobURL := job.URL
	if strings.HasPrefix(jobURL, "//") {
		jobURL = "https:" + jobURL
	}

	return &Job{
		Status:    JobStatus(job.Status),
		TTL:       job.TTL,
		Token:     job.Token,
		URL:       jobURL,
		createdAt: time.Now(),
		mu:        &sync.Mutex{},
	}, nil
}

// Send an image for classification. The image may be a os.File instance any
// other object implementing io.Reader interface. The params parameter is
// optional and may be nil.
//
// On success this method will immediately return a Job instance. Its status
// will initially be "not completed" as it usually takes 6-12 seconds for the
// server to process an image. In order to retrieve the annotation data, you
// need to keep updating the job status using the UpdateJob() method until the
// status changes. You may also use the WaitJob() method which does this
// automatically.
func (c *Client) ImageRequest(image io.Reader, filename string, params Params) (*Job, error) {
	buf := &bytes.Buffer{}
	multi := multipart.NewWriter(buf)

	field, err := multi.CreateFormFile("image_request[image]", filename)
	if err != nil {
		return nil, err
	}

	if _, err = io.Copy(field, image); err != nil {
		return nil, err
	}

	if params == nil {
		params = Params{}
	}

	if _, ok := params["image_request[locale]"]; !ok {
		params["image_request[locale]"] = DefaultLocale
	}

	for k, v := range params {
		multi.WriteField(k, v)
	}

	if err = multi.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", requestsURL, buf)
	if err != nil {
		return nil, err
	}

	hdr := req.Header
	hdr["User-Agent"] = userAgent

	auth, err := c.getAuthHeader("POST", requestsURL, params)
	if err != nil {
		return nil, err
	}
	hdr["Authorization"] = []string{auth}

	hdr["Content-Length"] = []string{strconv.Itoa(buf.Len())}
	hdr["Content-Type"] = []string{multi.FormDataContentType()}

	return c.doImageRequest(req)
}

// Send an image for classification. The image will be retrieved from the URL
// specified. The params parameter is optional and may be nil.
//
// On success this method will immediately return a Job instance. Its status
// will initially be "not completed" as it usually takes 6-12 seconds for the
// server to process an image. In order to retrieve the annotation data, you
// need to keep updating the job status using the UpdateJob() method until the
// status changes. You may also use the WaitJob() method which does this
// automatically.
func (c *Client) RemoteImageRequest(url string, params Params) (*Job, error) {
	if params == nil {
		params = Params{}
	}

	if _, ok := params["image_request[locale]"]; !ok {
		params["image_request[locale]"] = DefaultLocale
	}

	params["image_request[remote_image_url]"] = url
	values := params.values()

	body := bytes.NewBufferString(values.Encode())
	req, err := http.NewRequest("POST", requestsURL, body)
	if err != nil {
		return nil, err
	}

	hdr := req.Header
	hdr["User-Agent"] = userAgent

	auth, err := c.getAuthHeader("POST", requestsURL, params)
	if err != nil {
		return nil, err
	}
	fmt.Println("auth", auth)
	hdr["Authorization"] = []string{auth}

	hdr["Content-Length"] = []string{strconv.Itoa(body.Len())}

	return c.doImageRequest(req)
}

func (c *Client) updateJobFromRequest(job *Job, req *http.Request) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP error: %s", err)
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var updatedJob apiResponse

	if err := decoder.Decode(&updatedJob); err != nil {
		return fmt.Errorf("JSON error: %s", err)
	}

	if updatedJob.Error != nil {
		return fmt.Errorf("api error: %s, status code: %d", string(updatedJob.Error), resp.StatusCode)
	}

	job.Categories = updatedJob.Categories
	job.Name = updatedJob.Name
	job.Status = JobStatus(updatedJob.Status)
	job.TTL = updatedJob.TTL
	job.SkipReason = SkipReason(updatedJob.Reason)
	return nil
}

// Contact the server and update the job status. This method does nothing if
// the status has already changed from "not completed".
//
// After a request has been submitted, it usually takes 6-12 seconds to receive
// a completed response. We recommend polling for a response every 1 second
// after a 4 second delay from the initial request, while the status is "not
// completed". WaitJob() method does this automatically.
func (c *Client) UpdateJob(job *Job) error {
	job.mu.Lock()
	defer job.mu.Unlock()

	if job.Status != StatusNotCompleted {
		return nil
	}

	url := responsesURL + job.Token
	req, _ := http.NewRequest("GET", url, nil)

	hdr := req.Header
	hdr["User-Agent"] = userAgent

	auth, err := c.getAuthHeader("GET", url, nil)
	if err != nil {
		return err
	}
	hdr["Authorization"] = []string{auth}

	return c.updateJobFromRequest(job, req)
}

// Repost the job if it has timed out (StatusTimeout).
//
// ErrInvalidRepostStatus will be returned if current job status is different
// than StatusTimeout.
func (c *Client) RepostJob(job *Job) error {
	if job.Status != StatusTimeout {
		return ErrInvalidRepostStatus
	}

	url := fmt.Sprintf("%s/%s/repost", requestsURL, job.Token)
	req, _ := http.NewRequest("POST", url, nil)

	hdr := req.Header
	hdr["User-Agent"] = userAgent
	hdr["Content-Length"] = []string{"0"}

	auth, err := c.getAuthHeader("POST", url, nil)
	if err != nil {
		return err
	}
	hdr["Authorization"] = []string{auth}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP error: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("error reposting job: %s, status: %d", string(body), resp.StatusCode)
	}

	io.Copy(ioutil.Discard, resp.Body)
	return c.UpdateJob(job)
}

// Wait for the job until it has been processed. This method will block for up
// to timeout seconds. After that ErrTimeout will be returned. If the timeout
// parameter is set to 0, WaitJob() will wait infinitely.
//
// This method will wait for 4 seconds after the initial request and then will
// call UpdateJob() method every second until the status changes.
func (c *Client) WaitJob(job *Job, timeout time.Duration) error {
	timeoutAt := time.Now().Add(timeout)

	waitUntil := job.createdAt.Add(pollMinWait)
	now := time.Now()

	if now.Before(waitUntil) {
		time.Sleep(waitUntil.Sub(now))
	}

	for {
		if timeout > 0 && time.Now().After(timeoutAt) {
			return ErrTimeout
		}

		if err := c.UpdateJob(job); err != nil {
			return err
		}

		if job.Status != StatusNotCompleted {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}
