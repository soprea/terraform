package http

import (
	"bytes"
	"errors"
	"fmt"
	//	"io"
	"log"
	//	"net/http"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

func (b *Backend) States() ([]string, error) {
	result := []string{backend.DefaultStateName}
	//addressURL, _ := url.Parse(b.address)
	/*
		resp, err := http.Get(b.address)
		log.Printf("[DEBUG] BACKEND Stefan States addressURL is: %s", b.address)

		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		// Read in the body
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, resp.Body); err != nil {
			return nil, fmt.Errorf("Failed to read remote state: %s", err)
		}
		log.Printf("[DEBUG] BACKEND Stefan States buf is: %s", buf)
		result = append(result, buf.String())
	*/
	return result, nil
}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}
	c := b.Client()

	return c.Delete()
}

func (b *Backend) State(name string) (state.State, error) {
	log.Printf("[DEBUG] BACKEND Stefan State name is: %s", name)

	if name == "" {
		return nil, errors.New("missing state name")
	}
	if name == "bar" || name == "foo" {
		log.Printf("[DEBUG] BACKEND Stefan State client name is: %s", name)
	}
	client := &RemoteClient{
		client:        b.client,
		address:       b.statePath(name),
		updateMethod:  b.updateMethod,
		lockAddress:   b.lockAddress,
		unlockAddress: b.unlockAddress,
		lockMethod:    b.lockMethod,
		unlockMethod:  b.unlockMethod,
		username:      b.username,
		password:      b.password,
	}
	stateMgr := &remote.State{Client: client}
	// take a lock on this state while we write it
	lockInfo := state.NewLockInfo()
	lockInfo.Operation = "init"
	lockId, err := client.Lock(lockInfo)

	if err != nil {
		return nil, fmt.Errorf("failed to lock http state: %s", err)
	}

	// Local helper function so we can call it multiple places
	lockUnlock := func(parent error) error {
		if err := stateMgr.Unlock(lockId); err != nil {
			return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
		}
		return parent
	}

	// Grab the value
	if err := stateMgr.RefreshState(); err != nil {
		err = lockUnlock(err)
		return nil, err
	}

	// If we have no state, we have to create an empty state
	if v := stateMgr.State(); v == nil {

		if err := stateMgr.WriteState(terraform.NewState()); err != nil {
			err = lockUnlock(err)
			return nil, err
		}
		if err := stateMgr.PersistState(); err != nil {
			err = lockUnlock(err)
			return nil, err
		}

	}

	// Unlock, the state should now be initialized
	if err := lockUnlock(nil); err != nil {
		return nil, err
	}

	return stateMgr, nil
}

func (b *Backend) statePath(name string) string {
	if name == backend.DefaultStateName {
		paths := []string{b.address, "/default.tfstate"}
		var buf bytes.Buffer
		for _, p := range paths {
			buf.WriteString(p)
		}
		path := buf.String()

		return path
	}
	paths := []string{b.address, "/", name, ".tfstate"}
	var buf bytes.Buffer
	for _, p := range paths {
		buf.WriteString(p)
	}
	path := buf.String()

	return path

}

func (b *Backend) Client() *RemoteClient {
	return &RemoteClient{}
}

const errStateUnlock = `
Error unlocking http state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`
