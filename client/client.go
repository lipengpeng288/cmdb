// Copyright © 2018 Alfred Chou <unioverlord@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	yaml "github.com/ghodss/yaml"
	apis "github.com/universonic/cmdb/shared/apis"
	genericStorage "github.com/universonic/cmdb/shared/storage/generic"
	cliutil "github.com/universonic/cmdb/utils/cli"
	fsutil "github.com/universonic/cmdb/utils/filesystem"
)

const (
	jsonContentType = "application/json"
)

// Client is the main implementation of CMDB CLI client
type Client struct {
	addr       string
	httpClient *http.Client
}

// CreateMachine creates machine with given params
func (in *Client) CreateMachine(name, sshAddr string, sshPort uint16, sshUser, ipmiAddr, ipmiUser, ipmiPassword string, yamled bool) error {
	cv := genericStorage.NewMachine()
	cv.Name = name
	cv.SSHAddress = sshAddr
	cv.SSHUser = sshUser
	cv.SSHPort = sshPort
	cv.IPMIAddress = ipmiAddr
	cv.IPMIUser = ipmiUser
	cv.IPMIPassword = ipmiPassword
	return in.createMachine(cv, yamled)
}

// CreateMachineFromYAML creates machine with given YAML
func (in *Client) CreateMachineFromYAML(filename string, yamled bool) error {
	cv, err := in.parseMachineFromYAML(filename)
	if err != nil {
		return err
	}
	return in.createMachine(cv, yamled)
}

func (in *Client) createMachine(cv *genericStorage.Machine, yamled bool) error {
	dAtA, err := json.Marshal(cv)
	if err != nil {
		return err
	}
	resp, err := in.Post(apis.MachineAPI, jsonContentType, bytes.NewReader(dAtA))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dAtA, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", dAtA)
	}
	if yamled {
		dAtA, err = yaml.JSONToYAML(dAtA)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "%s\n", dAtA)
		return nil
	}
	err = json.Unmarshal(dAtA, cv)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	WriteASCIITable(&buf, cv.Header(), cv.Row())
	fmt.Fprintf(os.Stdout, "%s\n", buf.Bytes())
	return nil
}

// UpdateMachine updates machine with given params
func (in *Client) UpdateMachine(name, sshAddr string, sshPort uint16, sshUser, ipmiAddr, ipmiUser, ipmiPassword string, yamled bool) error {
	cv := genericStorage.NewMachine()
	cv.Name = name
	cv.SSHAddress = sshAddr
	cv.SSHUser = sshUser
	cv.SSHPort = sshPort
	cv.IPMIAddress = ipmiAddr
	cv.IPMIUser = ipmiUser
	cv.IPMIPassword = ipmiPassword
	return in.updateMachine(cv, yamled)
}

// UpdateMachineFromYAML updates machine with given YAML
func (in *Client) UpdateMachineFromYAML(filename string, yamled bool) error {
	cv, err := in.parseMachineFromYAML(filename)
	if err != nil {
		return err
	}
	return in.updateMachine(cv, yamled)
}

func (in *Client) updateMachine(cv *genericStorage.Machine, yamled bool) error {
	dAtA, err := json.Marshal(cv)
	if err != nil {
		return err
	}
	resp, err := in.Put(apis.MachineAPI, jsonContentType, bytes.NewReader(dAtA))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dAtA, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", dAtA)
	}
	if yamled {
		dAtA, err = yaml.JSONToYAML(dAtA)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "%s\n", dAtA)
		return nil
	}
	err = json.Unmarshal(dAtA, cv)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	WriteASCIITable(&buf, cv.Header(), cv.Row())
	fmt.Fprintf(os.Stdout, "%s\n", buf.Bytes())
	return nil
}

// GetMachine retrives a list of machine with given params
func (in *Client) GetMachine(name []string, yamled bool) error {
	var suffix string
	if len(name) == 0 {
		suffix = "*"
	} else {
		suffix = strings.Join(name, ",")
	}
	resp, err := in.Get(apis.MachineAPI + "?search=" + suffix)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dAtA, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", dAtA)
	}
	if yamled {
		dAtA, err = yaml.JSONToYAML(dAtA)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "%s\n", dAtA)
		return nil
	}
	ex := genericStorage.NewMachine()
	cv := genericStorage.NewMachineList()
	err = json.Unmarshal(dAtA, &cv.Members)
	if err != nil {
		return err
	}
	var (
		buf  bytes.Buffer
		rows [][]string
	)
	for _, each := range cv.Members {
		rows = append(rows, each.Row())
	}
	WriteASCIITable(&buf, ex.Header(), rows...)
	fmt.Fprintf(os.Stdout, "%s\n", buf.Bytes())
	return nil
}

// RemoveMachine deletes machine
func (in *Client) RemoveMachine(name string, yes bool) error {
	if !yes {
		prompt := cliutil.NewComfirmPrompt(os.Stderr, os.Stdin)
		if !prompt.Enforce() {
			return fmt.Errorf("User Abort")
		}
	}
	resp, err := in.Delete(apis.MachineAPI + "?target=" + name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		dAtA, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", dAtA)
	}
	fmt.Fprintf(os.Stderr, "Done.\n")
	return nil
}

// CreateMachineDigest generates digest on backend
func (in *Client) CreateMachineDigest() error {
	resp, err := in.Post(apis.MachineDigestAPI, "plain/text", bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		dAtA, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", dAtA)
	}
	fmt.Fprintf(os.Stderr, "Creating machine digest. You can wait for a while to check it later.\n")
	return nil
}

// ListMachineDigest retrieves a list machine digests by creation date.
func (in *Client) ListMachineDigest(id ...string) (err error) {
	var suffix string
	if len(id) == 0 {
		suffix = "?search=*"
	} else {
		suffix = fmt.Sprintf("?search=%s", id[0])
	}
	resp, err := in.Get(apis.MachineDigestAPI + suffix)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		dAtA, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", dAtA)
	}
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return err
	}
	cv := genericStorage.NewMachineDigestList()
	err = json.Unmarshal(buf.Bytes(), &cv.Members)
	if err != nil {
		return err
	}
	ex := genericStorage.NewMachineDigest()
	buf.Reset()
	var rows [][]string
	for _, each := range cv.Members {
		rows = append(rows, each.Row())
	}
	WriteASCIITable(&buf, ex.Header(), rows...)
	fmt.Fprintf(os.Stdout, "%s\n", buf.Bytes())
	return nil
}

// GetMachineDigest retrieve a digest from backend with given params
func (in *Client) GetMachineDigest(machine, date, output string, jsoned bool) (err error) {
	if output == "" {
		return fmt.Errorf("Output file must be specified")
	}
	var format, suffix string
	if date != "" {
		if jsoned {
			format = "json"
		} else {
			format = "xlsx"
		}
		var t time.Time
		t, err = time.Parse(time.RFC3339, date)
		if err != nil {
			return err
		}
		suffix = fmt.Sprintf("?scope=date&target=%d&format=%s", t.Unix(), format)
	} else if machine != "" {
		suffix = fmt.Sprintf("?scope=machine&target=%s&format=json", machine)
	} else {
		suffix = "?scope=date"
		if jsoned {
			suffix += "&format=json"
		} else {
			suffix += "&format=xlsx"
		}
	}
	exists, _ := fsutil.FileExists(output)
	if exists {
		return fmt.Errorf("File exists: %s", output)
	}
	fi, err := os.Create(output)
	if err != nil {
		return err
	}
	defer fi.Close()
	spinner := cliutil.NewSpinner()
	spinner.Prefix = "Awaiting generated result: "
	defer func() {
		spinner.Stop()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\r"+spinner.Prefix+"Failed!\n")
		} else {
			fmt.Fprintf(os.Stderr, "\r"+spinner.Prefix+"Completed!\n")
		}
	}()
	spinner.Start()
	resp, err := in.Get(apis.MachineDigestAPI + suffix)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		dAtA, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", dAtA)
	}
	_, err = io.Copy(fi, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func (in *Client) parseMachineFromYAML(filename string) (*genericStorage.Machine, error) {
	fi, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, fi)
	if err != nil {
		return nil, err
	}
	dAtA, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return nil, err
	}
	cv := genericStorage.NewMachine()
	err = json.Unmarshal(dAtA, cv)
	if err != nil {
		return nil, err
	}
	return cv, nil
}

func (in *Client) CreateAutoDiscovery(name, cidr string) (err error) {
	cv := genericStorage.NewAutoDiscovery()
	cv.SetName(name)
	cv.CIDR = cidr
	dAtA, err := json.Marshal(cv)
	if err != nil {
		return err
	}
	resp, err := in.Post(apis.AutoDiscoveryAPI, jsonContentType, bytes.NewReader(dAtA))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dAtA, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", dAtA)
	}
	err = json.Unmarshal(dAtA, cv)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	WriteASCIITable(&buf, cv.Header(), cv.Row())
	fmt.Fprintf(os.Stdout, "%s\n", buf.Bytes())
	return nil
}

func (in *Client) UpdateAutoDiscovery(name, cidr string) (err error) {
	cv := genericStorage.NewAutoDiscovery()
	cv.SetName(name)
	cv.CIDR = cidr
	dAtA, err := json.Marshal(cv)
	if err != nil {
		return err
	}
	resp, err := in.Put(apis.AutoDiscoveryAPI, jsonContentType, bytes.NewReader(dAtA))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dAtA, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", dAtA)
	}
	err = json.Unmarshal(dAtA, cv)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	WriteASCIITable(&buf, cv.Header(), cv.Row())
	fmt.Fprintf(os.Stdout, "%s\n", buf.Bytes())
	return nil
}

func (in *Client) UpdateDiscoveredMachines() (err error) {
	resp, err := in.Post(apis.DiscoveredMachinesAPI, "plain/text", bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		dAtA, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", dAtA)
	}
	fmt.Fprintf(os.Stderr, "Now we are refreshing the discovered machines list. You can wait for a while to check it later.\n")
	return nil
}

func (in *Client) GetDiscovertedMachines() (err error) {
	resp, err := in.Get(apis.DiscoveredMachinesAPI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dAtA, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", dAtA)
	}
	cv := genericStorage.NewDiscoveredMachines()
	err = json.Unmarshal(dAtA, cv)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	WriteASCIITable(&buf, cv.Header(), cv.Row())
	fmt.Fprintf(os.Stdout, "%s\n", buf.Bytes())
	return nil
}

func (in *Client) GetAutoDiscovery(name []string) (err error) {
	var suffix string
	if len(name) == 0 {
		suffix = "*"
	} else {
		suffix = strings.Join(name, ",")
	}
	resp, err := in.Get(apis.AutoDiscoveryAPI + "?search=" + suffix)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dAtA, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", dAtA)
	}
	ex := genericStorage.NewAutoDiscovery()
	cv := genericStorage.NewAutoDiscoveryList()
	err = json.Unmarshal(dAtA, &cv.Members)
	if err != nil {
		return err
	}
	var (
		buf  bytes.Buffer
		rows [][]string
	)
	for _, each := range cv.Members {
		rows = append(rows, each.Row())
	}
	WriteASCIITable(&buf, ex.Header(), rows...)
	fmt.Fprintf(os.Stdout, "%s\n", buf.Bytes())
	return nil
}

func (in *Client) RemoveAutoDiscovery(name string, yes bool) error {
	if !yes {
		prompt := cliutil.NewComfirmPrompt(os.Stderr, os.Stdin)
		if !prompt.Enforce() {
			return fmt.Errorf("User Abort")
		}
	}
	resp, err := in.Delete(apis.AutoDiscoveryAPI + "?target=" + name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		dAtA, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", dAtA)
	}
	fmt.Fprintf(os.Stderr, "Done.\n")
	return nil
}

// Post implements a generic POSTer
func (in *Client) Post(path, contentType string, body io.Reader) (*http.Response, error) {
	return in.request("POST", path, contentType, body)
}

// Put implements a generic PUTer
func (in *Client) Put(path, contentType string, body io.Reader) (*http.Response, error) {
	return in.request("PUT", path, contentType, body)
}

// Get implements a generic GETter
func (in *Client) Get(path string) (*http.Response, error) {
	return in.request("GET", path, "", nil)
}

// Delete implements a generic DELETEr
func (in *Client) Delete(path string) (*http.Response, error) {
	return in.request("DELETE", path, "", nil)
}

func (in *Client) request(method, path, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("http://%s", in.addr)+apis.APIv1+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CMDB-CLI/v0.1-alpha")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", jsonContentType)
	}
	return in.httpClient.Do(req)
}

// NewClient returns a new Client with given endpoint on specified network.
func NewClient(network, endpoint string) (*Client, error) {
	switch network {
	case "unix":
		return &Client{
			addr: filepath.Base(endpoint),
			httpClient: &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyFromEnvironment,
					DialContext: (&unixDialer{
						unixAddr: endpoint,
						Dialer: net.Dialer{
							Timeout:   30 * time.Second,
							KeepAlive: 30 * time.Second,
						},
					}).DialContext,
					MaxIdleConns:          100,
					IdleConnTimeout:       90 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
				},
				Timeout: time.Second * 10,
			},
		}, nil
	case "tcp":
		return &Client{
			addr: endpoint,
			httpClient: &http.Client{
				Transport: http.DefaultTransport,
				Timeout:   time.Second * 10,
			},
		}, nil
	default:
		return nil, fmt.Errorf("Unknown network: %s", network)
	}
}

type unixDialer struct {
	net.Dialer
	unixAddr string
}

func (in *unixDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return net.Dial("unix", in.unixAddr)
}
