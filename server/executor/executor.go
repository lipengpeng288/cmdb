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

package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	genericStorage "github.com/universonic/cmdb/shared/storage/generic"
	intutil "github.com/universonic/cmdb/utils/integration"
	zap "go.uber.org/zap"
)

const timeoutPollInterval = 500 * time.Millisecond

// Executor is currently a direct worker of scheduler.
type Executor struct {
	timeout  int
	closeCh  chan struct{}
	clzSubCh []chan struct{}
	lock     sync.RWMutex
	observer chan *genericStorage.MachineDigest
	logger   *zap.SugaredLogger
	storage  genericStorage.Storage
}

// Prepare initialize inner storage and logger for server
func (in *Executor) Prepare(storage genericStorage.Storage, logger *zap.SugaredLogger) {
	in.storage = storage
	in.logger = logger
}

// Serve starts executor
func (in *Executor) Serve() error {
	defer func() {
		for i := range in.clzSubCh {
			close(in.clzSubCh[i])
		}
		in.clzSubCh = in.clzSubCh[:0]
	}()
LOOP:
	for {
		select {
		case <-in.closeCh:
			// Pending on here to wait for a graceful shutdown and block new requests.
			// It just pass through if it is not in busy state.
			// In common cases, it will not take a long time to return after it accepted
			// the shutdown signal, so we do not need a timeout context here.
			in.lock.Lock()
			defer in.lock.Unlock()
			break LOOP
		case digest := <-in.observer:
			go in.collect(digest)
		}
	}
	return nil
}

// Subscribe attach an external channel to be used for callback function when server exited.
func (in *Executor) Subscribe() <-chan struct{} {
	subscription := make(chan struct{}, 1)
	in.clzSubCh = append(in.clzSubCh, subscription)
	return subscription
}

// NotifyDigest should be called if there is a incoming digest that need to be fulfilled.
func (in *Executor) NotifyDigest(digest *genericStorage.MachineDigest) {
	in.observer <- digest
}

// Stop shutdown the executor
func (in *Executor) Stop() {
	close(in.closeCh)
}

func (in *Executor) collect(digest *genericStorage.MachineDigest) (err error) {
	defer in.logger.Sync()
	defer func() {
		if err != nil {
			digest.State = "FAILED"
			in.logger.Errorf("Failed to generate digest due to: %v", err)
		}
		in.storage.Update(digest)
		in.logger.Infof("Successfully generated digest: %s", digest.GetGUID())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(in.timeout)*time.Second)
	defer cancel()

	var (
		dataset []genericStorage.MachineSnapshot
		abort   bool
	)
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		var err error
		defer in.logger.Sync()
		defer close(errChan)
		defer func() {
			if r := recover(); r != nil {
				errChan <- r.(error)
				in.logger.Errorf("Unexpected error occured: %v", err)
			}
		}()
		defer func() {
			if err != nil {
				digest.State = "FAILED"
				in.storage.Update(digest)
			}
		}()

		digest.State = "IN-PROGRESS"
		err = in.storage.Update(digest)
		if err != nil {
			errChan <- err
			return
		}
		in.logger.Infof("Refreshing machine information...")
		list := genericStorage.NewMachineList()
		err = in.storage.List(list)
		if err != nil {
			errChan <- err
			return
		}

		var inventory string
		for _, each := range list.Members {
			var sshAddr string
			if each.SSHAddress == "" {
				sshAddr = each.Name
			} else {
				sshAddr = each.SSHAddress
			}
			inventory += fmt.Sprintf(
				"%s ansible_connection=\"smart\" ansible_host=\"%s\" ansible_port=%d ansible_user=\"%s\" ipmi_addr=\"%s\" ipmi_user=\"%s\" ipmi_pass=\"%s\" department=\"%s\" comment=\"%s\"\n",
				each.Name, sshAddr, each.SSHPort, each.SSHUser, each.IPMIAddress, each.IPMIUser, each.IPMIPassword,
				each.ExtraInfo.Department, each.ExtraInfo.Comment,
			)
		}

		fi, e := ioutil.TempFile("", "")
		if e != nil {
			errChan <- e
			return
		}
		defer os.Remove(fi.Name())
		defer fi.Close() // Close anyway

		buf := bytes.NewReader([]byte(inventory))
		_, err = io.Copy(fi, buf)
		if err != nil {
			errChan <- err
			return
		}
		inventoryFile := fi.Name()
		fi.Close()

		var (
			ssh      map[string][]byte
			ipmi     map[string][]byte
			result   = make(map[string]*AnsibleResultCarrier)
			resultCV = make(map[string][]byte)
			wg       sync.WaitGroup
			errs     []error
		)
		wg.Add(2)
		go func() {
			defer in.logger.Sync()
			defer wg.Done()
			sshz := intutil.NewAnsible("canonical", inventoryFile)
			e := sshz.Execute()
			if abort {
				return
			}
			if e != nil {
				in.logger.Error(e)
				in.logger.Debugf("Verbose stdout during executing ansible module '%s': %s", sshz.Module, sshz.Stdout)
				in.logger.Debugf("Verbose stderr during executing ansible module '%s': %s", sshz.Module, sshz.Stderr)
				errs = append(errs, e)
			}
			ssh = sshz.Result
		}()
		go func() {
			defer in.logger.Sync()
			defer wg.Done()
			ipmiz := intutil.NewAnsible("ipmi", inventoryFile)
			e := ipmiz.Execute()
			if abort {
				return
			}
			if e != nil {
				in.logger.Error(e)
				in.logger.Debugf("Verbose stdout during executing ansible module '%s': %s", ipmiz.Module, ipmiz.Stdout)
				in.logger.Debugf("Verbose stderr during executing ansible module '%s': %s", ipmiz.Module, ipmiz.Stderr)
				errs = append(errs, e)
			}
			ipmi = ipmiz.Result
		}()
		wg.Wait()

		if abort {
			return
		}
		if len(errs) != 0 {
			var es []string
			for i := range errs {
				es = append(es, errs[i].Error())
			}
		}
		if len(ssh) != len(ipmi) {
			errChan <- fmt.Errorf("Unexpected corrupted ansible process")
			return
		}
		for k, v := range ipmi {
			cv0 := NewAnsibleResultMergableUnit()
			err = json.Unmarshal(ssh[k], cv0)
			if err != nil {
				in.logger.Errorf("Could not parse canonical result as JSON due to: %v", err)
				errChan <- err
				return
			}
			cv1 := NewAnsibleResultMergableUnit()
			err = json.Unmarshal(v, cv1)
			if err != nil {
				in.logger.Errorf("Could not parse ipmi result as JSON due to: %v", err)
				errChan <- err
				return
			}
			for k, v := range cv0.AnsibleFacts {
				cv1.AnsibleFacts[k] = v
			}
			cv1.Changed = cv0.Changed && cv1.Changed
			var dAtA []byte
			dAtA, err = json.Marshal(cv1)
			if err != nil {
				in.logger.Errorf("Could not unparse merged result as JSON due to: %v", err)
				errChan <- err
				return
			}
			resultCV[k] = dAtA
		}
		for name, each := range resultCV {
			cv := NewAnsibleResultCarrier()
			err = json.Unmarshal(each, &cv)
			if err != nil {
				in.logger.Errorf("Could not parse merged result into result carrier due to: %v", err)
				errChan <- err
				return
			}
			result[name] = cv
		}

		for name, each := range result {
			machine := genericStorage.NewMachine()
			machine.SetName(name)
			err = in.storage.Get(machine)
			if err != nil {
				in.logger.Errorf("Could not retrieve machine due to: %v", err)
				errChan <- err
				return
			}
			dataset = append(dataset, *ParseAnsibleResult(each, machine))
		}
	}()

	ticker := time.NewTicker(timeoutPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			abort = true
			return ctx.Err()
		case e := <-errChan:
			if e != nil {
				return e
			}
			return in.saveToDigest(dataset, digest)
		case <-ticker.C:
			continue
		}
	}
}

func (in *Executor) saveToDigest(snapshots []genericStorage.MachineSnapshot, digest *genericStorage.MachineDigest) error {
	in.lock.Lock()
	defer in.lock.Unlock()

	for i := range snapshots {
		obj := &snapshots[i]
		err := in.storage.Create(obj)
		if err != nil {
			return fmt.Errorf("Could not commit changes into storage due to: %v", err)
		}
		digest.Members = append(digest.Members, obj.ObjectMeta)
	}
	digest.State = "COMPLETED"
	return in.storage.Update(digest)
}

// NewExecutor returns a new executor with given storage and logger.
func NewExecutor(timeout int) *Executor {
	return &Executor{
		timeout:  timeout,
		closeCh:  make(chan struct{}, 1),
		observer: make(chan *genericStorage.MachineDigest, 1),
	}
}

// ParseAnsibleResult parse given AnsibleResultCarrier into a MachineSnapshot
func ParseAnsibleResult(cv *AnsibleResultCarrier, override *genericStorage.Machine) (result *genericStorage.MachineSnapshot) {
	result = genericStorage.NewMachineSnapshot()
	result.Namespace = override.GetName()
	if cv.Distribution == "OpenBSD" {
		result.OS = fmt.Sprintf("%s %s", cv.Distribution, cv.DistributionRelease)
	} else {
		result.OS = fmt.Sprintf("%s %s", cv.Distribution, cv.DistributionVersion)
	}
	result.Department = cv.Department
	switch cv.VirtualizationRole {
	case "NA", "host", "":
		result.Type = "Physical"
	case "?":
		result.Type = "(Not Sure)"
	default:
		result.Type = "Virtual"
	}
	result.Comment = cv.Comment
	result.Manufacturer = cv.IPMIManufacturer
	result.Model = cv.IPMIModel
	result.SerialNumber = cv.IPMISerialNumber
	if cv.IPMISystemLocation != nil {
		result.Location.Datacenter = cv.IPMISystemLocation.Datacenter
		result.Location.RoomName = cv.IPMISystemLocation.RoomName
		result.Location.Asile = cv.IPMISystemLocation.Aisle
		result.Location.RackName = cv.IPMISystemLocation.RackName
		result.Location.RackSlot = cv.IPMISystemLocation.RackSlot
		result.Location.DeviceSize = cv.IPMISystemLocation.DeviceSize
	}
	cpuAllTheSame := true
	var (
		cpuModel             string
		cpuFreq              string
		cpuCores, cpuThreads uint
		cpuCount             uint
	)
	for _, each := range cv.IPMICPUs {
		if cpuModel == "" {
			cpuModel = each.Name
			cpuFreq = each.BaseClockSpeed
		}
		if each.Name != cpuModel {
			cpuAllTheSame = false
		}
		cpuCores += each.Cores
		cpuThreads += each.Threads
		cpuCount++
	}
	if !cpuAllTheSame {
		cpuModel += " (and others)"
	}
	result.CPU.Model = cpuModel
	result.CPU.BaseFreq = cpuFreq
	result.CPU.Count = cpuCount
	result.CPU.Cores = cpuCores
	result.CPU.Threads = cpuThreads
	result.Memory.PopulatedDIMMs = cv.IPMIPopulatedDIMMs
	result.Memory.MaximumDIMMs = cv.IPMIMaxDIMMs
	result.Memory.InstalledMemory = cv.IPMIMemoryInstalled
	for _, each := range cv.IPMIVirtualDisks {
		result.Storage.VirtualDisks = append(result.Storage.VirtualDisks, genericStorage.VirtualDisk{
			Description: each.Description,
			Layout:      each.Layout,
			MediaType:   each.MediaType,
			Name:        each.Name,
			Size:        each.Size,
			State:       each.State,
			Status:      each.Status,
		})
	}
	for _, each := range cv.IPMIPhysicalDisks {
		result.Storage.PhysicalDisks = append(result.Storage.PhysicalDisks, genericStorage.PhysicalDisk{
			Description:  each.Description,
			MediaType:    each.MediaType,
			Name:         each.Name,
			SerialNumber: each.SerialNumber,
			Size:         each.Size,
			State:        each.State,
			Status:       each.Status,
		})
	}
	if cv.DefaultIPv4 != nil {
		result.Network.PrimaryIPAddress = cv.DefaultIPv4.Address
	}
	result.Network.IPMIAddress = cv.IPMIAddress
	for k, v := range cv.Interfaces {
		intf := new(genericStorage.LogicalInterface)
		intf.Name = k
		intf.Type = v.Type
		switch intf.Type {
		case "bonding":
			for _, each := range v.Slaves {
				newMember := new(genericStorage.LogicalInterfaceMember)
				newMember.Name = each
				newMember.MACAddress = cv.Interfaces[each].MACAddress
				intf.Members = append(intf.Members, *newMember)
			}
			result.Network.LogicalInterfaces = append(result.Network.LogicalInterfaces, *intf)
		}
	}

	// Check non-nil fields and deep copy to result
	if override.ExtraInfo.OS != "" {
		result.OS = override.ExtraInfo.OS
	}
	if override.ExtraInfo.Type != "" {
		result.Type = override.ExtraInfo.Type
	}
	if override.ExtraInfo.Department != "" {
		result.Department = override.ExtraInfo.Department
	}
	if override.ExtraInfo.Comment != "" {
		result.Comment = override.ExtraInfo.Comment
	}
	if override.ExtraInfo.Location.Datacenter != "" {
		result.Location.Datacenter = override.ExtraInfo.Location.Datacenter
	}
	if override.ExtraInfo.Location.RoomName != "" {
		result.Location.RoomName = override.ExtraInfo.Location.RoomName
	}
	if override.ExtraInfo.Location.Asile != "" {
		result.Location.Asile = override.ExtraInfo.Location.Asile
	}
	if override.ExtraInfo.Location.RackName != "" {
		result.Location.RackName = override.ExtraInfo.Location.RackName
	}
	if override.ExtraInfo.Location.RackSlot != "" {
		result.Location.RackSlot = override.ExtraInfo.Location.RackSlot
	}
	if override.ExtraInfo.Location.DeviceSize != "" {
		result.Location.DeviceSize = override.ExtraInfo.Location.DeviceSize
	}
	if override.ExtraInfo.CPU.BaseFreq != "" {
		result.CPU.BaseFreq = override.ExtraInfo.CPU.BaseFreq
	}
	if override.ExtraInfo.CPU.Cores != 0 {
		result.CPU.Cores = override.ExtraInfo.CPU.Cores
	}
	if override.ExtraInfo.CPU.Count != 0 {
		result.CPU.Count = override.ExtraInfo.CPU.Count
	}
	if override.ExtraInfo.CPU.Model != "" {
		result.CPU.Model = override.ExtraInfo.CPU.Model
	}
	if override.ExtraInfo.CPU.Threads != 0 {
		result.CPU.Threads = override.ExtraInfo.CPU.Threads
	}
	if override.ExtraInfo.Memory.InstalledMemory != "" {
		result.Memory.InstalledMemory = override.ExtraInfo.Memory.InstalledMemory
	}
	if override.ExtraInfo.Memory.MaximumDIMMs != 0 {
		result.Memory.MaximumDIMMs = override.ExtraInfo.Memory.MaximumDIMMs
	}
	if override.ExtraInfo.Memory.PopulatedDIMMs != 0 {
		result.Memory.PopulatedDIMMs = override.ExtraInfo.Memory.PopulatedDIMMs
	}
	if len(override.ExtraInfo.Storage.PhysicalDisks) != 0 {
		result.Storage.PhysicalDisks = override.ExtraInfo.Storage.PhysicalDisks
	}
	if len(override.ExtraInfo.Storage.VirtualDisks) != 0 {
		result.Storage.VirtualDisks = override.ExtraInfo.Storage.VirtualDisks
	}
	if len(override.ExtraInfo.Network.LogicalInterfaces) != 0 {
		result.Network.LogicalInterfaces = override.ExtraInfo.Network.LogicalInterfaces
	}
	return
}
