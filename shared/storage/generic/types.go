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

package generic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const (
	// RESOURCE_MACHINE indicates the kind of a Machine
	RESOURCE_MACHINE = "machine"
	// RESOURCE_MACHINE_SNAPSHOT indicates the kind of a MachineSnapshot
	RESOURCE_MACHINE_SNAPSHOT = "machine_snapshot"
	// RESOURCE_MACHINE_DIGEST indicates the kind of a MachineDigest
	RESOURCE_MACHINE_DIGEST = "machine_digest"
)

// ObjectMeta is the metadata of objects
type ObjectMeta struct {
	// GUID as a unified identifier for the object
	GUID string `json:"guid,omitempty" protobuf:"bytes,1,req,name=guid"`
	// Kind represents the resource name of the object
	Kind string `json:"kind,omitempty" protobuf:"bytes,2,req,name=kind"`
	// Name is the name of the object
	Name string `json:"name,omitempty" protobuf:"bytes,3,req,name=name"`
	// Namespace is the name of the object. Generally it is the name of its owner.
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,name=namespace"`
	// CreatedAt indicates the creation timestamp
	CreatedAt Time `json:"created_at,omitempty" protobuf:"bytes,5,req,name=created_at"`
	// UpdatedAt indicates the updating timestamp
	UpdatedAt *Time `json:"updated_at,omitempty" protobuf:"bytes,6,opt,name=updated_at"`
	// Deleting indicates that the object is being deleted.
	Deleting bool `json:"deleting,omitempty" protobuf:"varint,7,opt,name=deleting"`
}

// SetGUID set the GUID for an object
func (in *ObjectMeta) SetGUID(id string) {
	in.GUID = id
}

// GetGUID returns the GUID of an object
func (in *ObjectMeta) GetGUID() string {
	return in.GUID
}

// SetKind set the Kind for an object
func (in *ObjectMeta) SetKind(kind string) {
	in.Kind = kind
}

// GetKind returns the Kind of an object
func (in *ObjectMeta) GetKind() string {
	return in.Kind
}

// SetName set the Name for an object
func (in *ObjectMeta) SetName(name string) {
	in.Name = name
}

// GetName returns the Name of an object
func (in *ObjectMeta) GetName() string {
	return in.Name
}

// SetNamespace set the Namespace for an object
func (in *ObjectMeta) SetNamespace(ns string) {
	in.Namespace = ns
}

// GetNamespace returns the Namespace of an object
func (in *ObjectMeta) GetNamespace() string {
	return in.Namespace
}

// SetCreationTimestamp set the CreatedAt for an object
func (in *ObjectMeta) SetCreationTimestamp(timestamp time.Time) {
	in.CreatedAt = Time{timestamp}
}

// GetCreationTimestamp returns the creation timestamp of an object
func (in *ObjectMeta) GetCreationTimestamp() time.Time {
	return in.CreatedAt.Time
}

// SetUpdatingTimestamp set the UpdatedAt for an object
func (in *ObjectMeta) SetUpdatingTimestamp(timestamp time.Time) {
	in.UpdatedAt = &Time{timestamp}
}

// GetUpdatingTimestamp returns the updating timestamp of an object
func (in *ObjectMeta) GetUpdatingTimestamp() *time.Time {
	if in.UpdatedAt == nil {
		return nil
	}
	return &in.UpdatedAt.Time
}

// IsDeleting indicates whether the object is being deleted
func (in *ObjectMeta) IsDeleting() bool {
	return in.Deleting
}

// HasNamespace returns true if object is namespace-sensitive. This could be overriden
// within specific object.
func (in *ObjectMeta) HasNamespace() bool { return false }

// ObjectListMeta is the metadata of object list
type ObjectListMeta struct {
	// Kind represents the resource name of the object
	Kind string `json:"kind,omitempty"`
	// Isolated represents it is a list of object with namespace
	Isolated bool `json:"isolated,omitempty"`
}

// GetKind returns the Kind of an object list
func (in *ObjectListMeta) GetKind() string {
	return in.Kind
}

// HasNamespace returns true if it has namespace
func (in *ObjectListMeta) HasNamespace() bool {
	return in.Isolated
}

// Time is a wrapper around time.Time which supports correct
// marshaling to YAML and JSON.  Wrappers are provided for many
// of the factory methods that the time package offers.
//
// +protobuf.options.marshal=false
// +protobuf.as=Timestamp
// +protobuf.options.(gogoproto.goproto_stringer)=false
type Time struct {
	time.Time `protobuf:"-"`
}

// Machine indicates machine data object
type Machine struct {
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	SSHAddress   string                 `json:"ssh_addr,omitempty" protobuf:"bytes,2,opt,name=ssh_addr"`
	SSHPort      uint16                 `json:"ssh_port,omitempty" protobuf:"bytes,3,opt,name=ssh_port"`
	SSHUser      string                 `json:"ssh_user,omitempty" protobuf:"bytes,4,opt,name=ssh_user"`
	IPMIAddress  string                 `json:"ipmi_addr,omitempty" protobuf:"bytes,5,req,name=ipmi_addr"`
	IPMIUser     string                 `json:"ipmi_user,omitempty" protobuf:"bytes,6,req,name=ipmi_user"`
	IPMIPassword string                 `json:"ipmi_pass,omitempty" protobuf:"bytes,7,req,name=ipmi_pass"`
	ExtraInfo    MachineOverridableInfo `json:"extra_info,omitempty" protobuf:"bytes,8,opt,name=extra_info"`
}

// Header returns a set of headers that will be used for generating ASCII table.
func (in *Machine) Header() []string {
	return []string{"GUID", "Name", "SSH Address", "SSH Port", "SSH User", "IPMI Address", "IPMI User", "Extra Info", "Created At", "Updated At"}
}

// Row returns the value of object as a row of ASCII table.
func (in *Machine) Row() (row []string) {
	extra, err := json.Marshal(in.ExtraInfo)
	if err != nil {
		panic(err)
	}
	row = append(row,
		in.GUID,
		in.Name,
		in.SSHAddress,
		strconv.Itoa(int(in.SSHPort)),
		in.SSHUser,
		in.IPMIAddress,
		in.IPMIUser,
		string(extra),
		in.CreatedAt.String(),
	)
	if in.GetUpdatingTimestamp() != nil && !in.GetUpdatingTimestamp().IsZero() {
		return append(row, in.UpdatedAt.String())
	}
	return append(row, "")
}

// NewMachine generates a new empty Machine instance
func NewMachine() *Machine {
	return &Machine{
		ObjectMeta: ObjectMeta{Kind: RESOURCE_MACHINE},
	}
}

// MachineList indicates list of Machine
type MachineList struct {
	ObjectListMeta `json:",inline"`
	Members        []Machine `json:"members,omitempty"`
}

// AppendRaw appends raw format data to object list, and returns any encountered error.
func (in *MachineList) AppendRaw(dAtA []byte) error {
	cv := NewMachine()
	if err := json.Unmarshal(dAtA, cv); err != nil {
		return err
	}
	in.Members = append(in.Members, *cv)
	return nil
}

// NewMachineList generates a new empty MachineList instance
func NewMachineList() *MachineList {
	return &MachineList{
		ObjectListMeta: ObjectListMeta{
			Kind: RESOURCE_MACHINE,
		},
	}
}

// MachineSnapshot represents a dataset of state of a single machine at a specific time
type MachineSnapshot struct {
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Manufacturer           string `json:"manufacturer,omitempty" protobuf:"bytes,2,opt,name=manufacturer"`
	Model                  string `json:"model,omitempty" protobuf:"bytes,3,opt,name=model"`
	SerialNumber           string `json:"serial_number,omitempty" protobuf:"bytes,4,opt,name=serial_number"`
	MachineOverridableInfo `json:",inline"`
}

// Header returns a set of headers that will be used for generating ASCII table.
func (in *MachineSnapshot) Header() []string {
	return []string{"GUID", "Name", "OS", "Type", "Department", "Comment", "Manufacturer", "Model", "Serial Num",
		"Datacenter", "Room", "Aisle", "Rack Name", "Rack Slot", "Device Size", "CPU Model", "CPU Base Freq",
		"CPU Count", "CPU Cores", "Installed Memory", "Populated DIMMs", "Virtual Disks", "Physical Disks",
		"Primary IP", "IPMI Address", "Logical Interfaces", "Created At",
	}
}

// Row returns the value of object as a row of ASCII table.
func (in *MachineSnapshot) Row() (row []string) {
	vds, err := json.Marshal(in.Storage.VirtualDisks)
	if err != nil {
		panic(err)
	}
	pds, err := json.Marshal(in.Storage.PhysicalDisks)
	if err != nil {
		panic(err)
	}
	lis, err := json.Marshal(in.Network.LogicalInterfaces)
	if err != nil {
		panic(err)
	}
	return append(row,
		in.GUID,
		in.Name,
		in.OS,
		in.Type,
		in.Department,
		in.Comment,
		in.Manufacturer,
		in.Model,
		in.SerialNumber,
		in.Location.Datacenter,
		in.Location.RoomName,
		in.Location.Aisle,
		in.Location.RackName,
		in.Location.RackSlot,
		in.Location.DeviceSize,
		in.CPU.Model,
		in.CPU.BaseFreq,
		fmt.Sprintf("%d", in.CPU.Count),
		fmt.Sprintf("%d / %d", in.CPU.Cores, in.CPU.Threads),
		in.Memory.InstalledMemory,
		fmt.Sprintf("%d / %d", in.Memory.PopulatedDIMMs, in.Memory.MaximumDIMMs),
		string(vds),
		string(pds),
		in.Network.PrimaryIPAddress,
		in.Network.IPMIAddress,
		string(lis),
		in.CreatedAt.String(),
	)
}

// HasNamespace returns true if object is namespace-sensitive
func (in *MachineSnapshot) HasNamespace() bool { return true }

// NewMachineSnapshot generates a new empty MachineSnapshot instance
func NewMachineSnapshot() *MachineSnapshot {
	return &MachineSnapshot{
		ObjectMeta: ObjectMeta{Kind: RESOURCE_MACHINE_SNAPSHOT},
	}
}

// MachineOverridableInfo indicates the changable part of information of a machine
type MachineOverridableInfo struct {
	OS         string             `json:"os,omitempty" protobuf:"bytes,1,opt,name=os"`
	Type       string             `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`
	Department string             `json:"department,omitempty" protobuf:"bytes,3,opt,name=department"`
	Comment    string             `json:"comment,omitempty" protobuf:"bytes,4,opt,name=comment"`
	Location   MachineLocation    `json:"location,omitempty" protobuf:"bytes,5,opt,name=location"`
	CPU        MachineCPUInfo     `json:"cpu,omitempty" protobuf:"bytes,6,opt,name=cpu"`
	Memory     MachineMemInfo     `json:"memory,omitempty" protobuf:"bytes,7,opt,name=memory"`
	Storage    MachineStorageInfo `json:"storage,omitempty" protobuf:"bytes,8,opt,name=storage"`
	Network    MachineNetworkInfo `json:"network,omitempty" protobuf:"bytes,9,opt,name=network"`
}

// MachineLocation indicates a machine's location info.
type MachineLocation struct {
	Datacenter string `json:"datacenter,omitempty" protobuf:"bytes,1,opt,name=datacenter"`
	RoomName   string `json:"room_name,omitempty" protobuf:"bytes,2,opt,name=room_name"`
	Aisle      string `json:"aisle,omitempty" protobuf:"bytes,3,opt,name=aisle"`
	RackName   string `json:"rack_name,omitempty" protobuf:"bytes,4,opt,name=rack_name"`
	RackSlot   string `json:"rack_slot,omitempty" protobuf:"bytes,5,opt,name=rack_slot"`
	DeviceSize string `json:"device_size,omitempty" protobuf:"bytes,6,opt,name=device_size"`
}

// MachineCPUInfo indicates a machine's CPU info.
type MachineCPUInfo struct {
	Model    string `json:"model,omitempty" protobuf:"bytes,1,opt,name=model"`
	BaseFreq string `json:"base_freq,omitempty" protobuf:"bytes,2,opt,name=base_freq"`
	Count    uint   `json:"count,omitempty" protobuf:"varint,3,opt,name=count"`
	Cores    uint   `json:"cores,omitempty" protobuf:"varint,4,opt,name=cores"`
	Threads  uint   `json:"threads,omitempty" protobuf:"varint,5,opt,name=threads"`
}

// MachineMemInfo indicates a machine's memory info.
type MachineMemInfo struct {
	InstalledMemory string `json:"installed_memory,omitempty" protobuf:"bytes,1,opt,name=installed_memory"`
	PopulatedDIMMs  uint   `json:"populated_dimms,omitempty" protobuf:"varint,2,opt,name=populated_dimms"`
	MaximumDIMMs    uint   `json:"maximum_dimms,omitempty" protobuf:"varint,3,opt,name=maximum_dimms"`
}

// MachineStorageInfo indicates a machine's storage info.
type MachineStorageInfo struct {
	VirtualDisks  []VirtualDisk  `json:"virtual_disks,omitempty" protobuf:"bytes,1,rep,name=virtual_disks"`
	PhysicalDisks []PhysicalDisk `json:"physical_disks,omitempty" protobuf:"bytes,2,rep,name=physical_disks"`
}

// VirtualDisk indicates the information of a virtual disk
type VirtualDisk struct {
	Description string `json:"description,omitempty,omitempty" protobuf:"bytes,1,opt,name=description"`
	Layout      string `json:"layout,omitempty,omitempty" protobuf:"bytes,2,opt,name=layout"`
	MediaType   string `json:"media_type,omitempty,omitempty" protobuf:"bytes,3,opt,name=media_type"`
	Name        string `json:"name,omitempty,omitempty" protobuf:"bytes,4,opt,name=name"`
	Size        string `json:"size,omitempty,omitempty" protobuf:"bytes,5,opt,name=size"`
	State       string `json:"state,omitempty,omitempty" protobuf:"bytes,6,opt,name=state"`
	Status      string `json:"status,omitempty,omitempty" protobuf:"bytes,7,opt,name=status"`
}

// PhysicalDisk indicates the information of a physical disk
type PhysicalDisk struct {
	Description  string `json:"description,omitempty" protobuf:"bytes,1,opt,name=description"`
	MediaType    string `json:"media_type,omitempty" protobuf:"bytes,2,opt,name=media_type"`
	Name         string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
	SerialNumber string `json:"serial_number,omitempty" protobuf:"bytes,4,opt,name=serial_number"`
	Size         string `json:"size,omitempty" protobuf:"bytes,5,opt,name=size"`
	State        string `json:"state,omitempty" protobuf:"bytes,6,opt,name=state"`
	Status       string `json:"status,omitempty" protobuf:"bytes,7,opt,name=status"`
}

// MachineNetworkInfo indicates a machine's network info.
type MachineNetworkInfo struct {
	PrimaryIPAddress  string             `json:"primary_ip_address,omitempty" protobuf:"bytes,1,opt,name=primary_ip_address"`
	IPMIAddress       string             `json:"ipmi_address,omitempty" protobuf:"bytes,2,opt,name=ipmi_address"`
	LogicalInterfaces []LogicalInterface `json:"logical_intfs,omitempty" protobuf:"bytes,3,rep,name=logical_intfs"`
}

// LogicalInterface indicates the information of a logical network interface such as bonding.
type LogicalInterface struct {
	Name    string                   `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Type    string                   `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`
	Members []LogicalInterfaceMember `json:"members,omitempty" protobuf:"bytes,3,rep,name=members"`
}

// LogicalInterfaceMember indicates the information of a member interface inside a logical interface.
type LogicalInterfaceMember struct {
	Name       string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	MACAddress string `json:"mac_address,omitempty" protobuf:"bytes,2,opt,name=mac_address"`
}

// MachineSnapshotList indicates list of MachineSnapshot
type MachineSnapshotList struct {
	ObjectListMeta `json:",inline"`
	Members        []MachineSnapshot `json:"members,omitempty"`
}

// AppendRaw appends raw format data to object list, and returns any encountered error.
func (in *MachineSnapshotList) AppendRaw(dAtA []byte) error {
	cv := NewMachineSnapshot()
	if err := json.Unmarshal(dAtA, cv); err != nil {
		return err
	}
	in.Members = append(in.Members, *cv)
	return nil
}

// NewMachineSnapshotList generates a new empty MachineSnapshotList instance
func NewMachineSnapshotList() *MachineSnapshotList {
	return &MachineSnapshotList{
		ObjectListMeta: ObjectListMeta{
			Kind:     RESOURCE_MACHINE_SNAPSHOT,
			Isolated: true,
		},
	}
}

// MachineDigest represents a set of MachineSnapshot at a specific date, which is
// used for generating report. Its GUID restrictly matches its Name.
type MachineDigest struct {
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	State   string       `json:"state,omitempty" protobuf:"bytes,2,opt,name=state"`
	Members []ObjectMeta `json:"members,omitempty" protobuf:"bytes,3,rep,name=members"`
}

// Header returns a set of headers that will be used for generating ASCII table.
func (in *MachineDigest) Header() []string {
	return []string{"GUID", "State", "Machines", "Created At", "Updated At"}
}

// Row returns the value of object as a row of ASCII table.
func (in *MachineDigest) Row() []string {
	row := []string{
		in.GetGUID(),
		in.State,
		fmt.Sprintf("%d", len(in.Members)),
		in.GetCreationTimestamp().String(),
	}
	if in.GetUpdatingTimestamp() != nil && !in.GetUpdatingTimestamp().IsZero() {
		return append(row, in.UpdatedAt.String())
	}
	return append(row, "")
}

// NewMachineDigest generates a new empty MachineDigest instance
func NewMachineDigest() *MachineDigest {
	return &MachineDigest{
		ObjectMeta: ObjectMeta{Kind: RESOURCE_MACHINE_DIGEST},
	}
}

// MachineDigestList indicates list of MachineDigest
type MachineDigestList struct {
	ObjectListMeta `json:",inline"`
	Members        []MachineDigest `json:"members,omitempty"`
}

// AppendRaw appends raw format data to object list, and returns any encountered error.
func (in *MachineDigestList) AppendRaw(dAtA []byte) error {
	cv := NewMachineDigest()
	if err := json.Unmarshal(dAtA, cv); err != nil {
		return err
	}
	in.Members = append(in.Members, *cv)
	return nil
}

// NewMachineDigestList generates a new empty MachineDigestList instance
func NewMachineDigestList() *MachineDigestList {
	return &MachineDigestList{
		ObjectListMeta: ObjectListMeta{
			Kind: RESOURCE_MACHINE_DIGEST,
		},
	}
}

type HealthCheck struct {
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	CIDR HealthCheckCIDR `json:"cidr,omitempty" protobuf:"bytes,2,opt,name=cidr"`
}

type HealthCheckCIDR struct {
	IP   string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`
	Mask string `json:"mask,omitempty" protobuf:"bytes,2,opt,name=mask"`
}

type Healthz struct {
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Existing []HealthzExistingMachine `json:"existing,omitempty" protobuf:"bytes,2,rep,name=existing"`
	Found    []HealthzUnknownMachine  `json:"found,omitempty" protobuf:"bytes,3,rep,name=found"`
}

type HealthzExistingMachine struct {
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	IP   string `json:"ip,omitempty" protobuf:"bytes,2,opt,name=ip"`
	OK   bool   `json:"ok,omitempty" protobuf:"varint,3,opt,name=ok"`
}

type HealthzUnknownMachine struct {
	IP    string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`
	Ports []int  `json:"ports,omitempty" protobuf:"bytes,2,rep,name=ports"`
}
