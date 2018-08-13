package http

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

// remoteClient is used by "state/remote".State to read and write
// blobs representing state.
// Implements "state/remote".ClientLocker
type RemoteClient struct {
	client *http.Client

	// The fields below are set from configure
	address       string
	updateMethod  string
	lockAddress   string
	unlockMethod  string
	lockMethod    string
	unlockAddress string
	username      string
	password      string

	lockID       string
	jsonLockInfo []byte
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	// Convert address to type URL
	addressURL, _ := url.Parse(c.address)
	resp, err := c.httpRequest("GET", addressURL, nil, "get state")
	log.Printf("[DEBUG] Client Stefan GET addressURL is: %s", addressURL)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle the common status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Handled after
	case http.StatusNoContent:
		return nil, nil
	case http.StatusNotFound:
		return nil, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("HTTP remote state endpoint requires auth")
	case http.StatusForbidden:
		return nil, fmt.Errorf("HTTP remote state endpoint invalid auth")
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("HTTP remote state internal server error")
	default:
		return nil, fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}

	// Read in the body
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}
	log.Printf("[DEBUG] Client Stefan buf is: %s", buf)

	// Create the payload
	payload := &remote.Payload{
		Data: buf.Bytes(),
	}
	log.Printf("[DEBUG] Client Stefan payload is: %s", payload)

	// If there was no data, then return nil
	//if buf == nil || len(buf.Bytes()) == 0 {
	//	return nil, fmt.Errorf("[DEBUG] State %s has no data.", addressURL)
	//}

	md5 := md5.Sum(buf.Bytes())
	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	// Check for the MD5
	if raw := resp.Header.Get("Content-MD5"); raw != "" {
		md5, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf(
				"Failed to decode Content-MD5 '%s': %s", raw, err)
		}

		payload.MD5 = md5
	} else {
		// Generate the MD5
		hash := md5
		payload.MD5 = hash[:]
	}

	return payload, nil
}

func (c *RemoteClient) Put(data []byte) error {
	// Copy the target URL
	addressURL, _ := url.Parse(c.address)
	log.Printf("[DEBUG] Client Stefan PUT addressURL is: %s", addressURL)

	base := *addressURL
	log.Printf("[DEBUG] Client Stefan PUT base is: %#v", base)

	if c.lockID != "" {
		query := base.Query()
		query.Set("ID", c.lockID)
		base.RawQuery = query.Encode()
	}

	var method = c.updateMethod
	resp, err := c.httpRequest(method, &base, &data, "upload state")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle the error codes
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
}

func (c *RemoteClient) Delete() error {
	// Make the request
	addressURL, _ := url.Parse(c.address)
	resp, err := c.httpRequest("DELETE", addressURL, nil, "delete state")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle the error codes
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
}

// Lock writes to a lock file, ensuring file creation. Returns the generation number, which must be passed to Unlock().
func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	lockURL, _ := url.Parse(c.lockAddress)
	c.lockID = ""

	jsonLockInfo := info.Marshal()
	resp, err := c.httpRequest(c.lockMethod, lockURL, &jsonLockInfo, "lock")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		c.lockID = info.ID
		c.jsonLockInfo = jsonLockInfo
		return info.ID, nil
	case http.StatusUnauthorized:
		return "", fmt.Errorf("HTTP remote state endpoint requires auth")
	case http.StatusForbidden:
		return "", fmt.Errorf("HTTP remote state endpoint invalid auth")
	case http.StatusConflict, http.StatusLocked:
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("HTTP remote state already locked, failed to read body")
		}
		existing := state.LockInfo{}
		err = json.Unmarshal(body, &existing)
		if err != nil {
			return "", fmt.Errorf("HTTP remote state already locked, failed to unmarshal body")
		}
		return "", fmt.Errorf("HTTP remote state already locked: ID=%s", existing.ID)
	default:
		return "", fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}

func (c *RemoteClient) Unlock(id string) error {
	unlockURL, _ := url.Parse(c.unlockAddress)
	resp, err := c.httpRequest(c.unlockMethod, unlockURL, &c.jsonLockInfo, "unlock")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}

func (c *RemoteClient) httpRequest(method string, url *url.URL, data *[]byte, what string) (*http.Response, error) {
	// If we have data we need a reader
	var reader io.Reader = nil
	if data != nil {
		reader = bytes.NewReader(*data)
	}
	// Create the request
	req, err := http.NewRequest(method, url.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to make %s HTTP request: %s", what, err)
	}
	// Setup basic auth
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Work with data/body
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(*data))

		// Generate the MD5
		hash := md5.Sum(*data)
		b64 := base64.StdEncoding.EncodeToString(hash[:])
		req.Header.Set("Content-MD5", b64)
	}

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to %s: %v", what, err)
	}

	return resp, nil
}
