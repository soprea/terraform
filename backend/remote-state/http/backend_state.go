package http

import (
	"bytes"
	"errors"
	"fmt"
	//"io"
	"log"
	"sort"
	//	"net/http"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

const (
	stateFileSuffix = ".tfstate"
	lockFileSuffix  = ".tflock"
)

func (b *Backend) States() ([]string, error) {
	//result := []string{backend.DefaultStateName}
  var result []string
	client := &RemoteClient{
		client:        b.client,
		address:       b.address,
		updateMethod:  b.updateMethod,
		lockAddress:   b.lockAddress,
		unlockAddress: b.unlockAddress,
		lockMethod:    b.lockMethod,
		unlockMethod:  b.unlockMethod,
		username:      b.username,
		password:      b.password,
	}

	log.Printf("[DEBUG] BACKEND Stefan States client is: [%+v]", client)
	resp, err := client.Get()
	if err != nil {
		return nil, err
	}
	// Read in the body
	log.Printf("[DEBUG] BACKEND Stefan States resp.Data is: [%s]", string(resp.Data))

	buff := string(resp.Data)
	log.Printf("[DEBUG] BACKEND Stefan States buff is: [%s]", buff)
	buff = strings.Replace(buff, ",", " ", -1)
	buffSlice := strings.Fields(buff)
	log.Printf("[DEBUG] BACKEND Stefan States buff is: [%s]", buff)
	log.Printf("[DEBUG] BACKEND Stefan States buff type is: [%T]", buff)

	for _, el := range buffSlice {
		log.Printf("[DEBUG] BACKEND Stefan States obj is: [%s]", string(el))
		eltp := strings.TrimSuffix(el, ".tfstate")
		log.Printf("[DEBUG] BACKEND Stefan States obj is: [%s]", eltp)

		elts := strings.TrimPrefix(eltp, "/")
		log.Printf("[DEBUG] BACKEND Stefan States obj is: [%s]", elts)
		if elts != "" {
			result = append(result, elts)
		}
	}

	log.Printf("[DEBUG] BACKEND Stefan States result is: [%s]", result)
	sort.Strings(result[1:])

	return result, nil
}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}
	client, err := b.remoteClient(name)
	if err != nil {
		return err
	}

	return client.Delete()
}

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, errors.New("missing state name")
	}
	client := &RemoteClient{
		client:        b.client,
		address:       b.statePath(name),
		updateMethod:  b.updateMethod,
		lockAddress:   b.lockPath(name),
		unlockAddress: b.lockPath(name),
		lockMethod:    b.lockMethod,
		unlockMethod:  b.unlockMethod,
		username:      b.username,
		password:      b.password,
	}
	return client, nil
}

func (b *Backend) State(name string) (state.State, error) {
	client, err := b.remoteClient(name)
	if err != nil {
		return nil, err
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
	paths := []string{b.address, "/", name, stateFileSuffix}
	var buf bytes.Buffer
	for _, p := range paths {
		buf.WriteString(p)
	}
	path := buf.String()

	return path

}

func (b *Backend) lockPath(name string) string {
	paths := []string{b.address, "/", name, lockFileSuffix}
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
