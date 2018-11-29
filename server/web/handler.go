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

package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	excel "github.com/360EntSecGroup-Skylar/excelize"
	mux "github.com/gorilla/mux"
	apis "github.com/universonic/cmdb/shared/apis"
	genericStorage "github.com/universonic/cmdb/shared/storage/generic"
	zap "go.uber.org/zap"
)

// Handler is a bundled API handler
type Handler struct {
	*mux.Router
	wwwroot string
	storage genericStorage.Storage
	logger  *zap.SugaredLogger
}

// Frontend handles GET requests from /*
func (in *Handler) Frontend(w http.ResponseWriter, r *http.Request) {
	defer in.finalizeHeader(w)
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if ext := filepath.Ext(r.URL.Path); ext == "" {
		http.ServeFile(w, r, in.wwwroot)
		return
	}
	http.FileServer(http.Dir(in.wwwroot)).ServeHTTP(w, r)
}

// Machine handles requests from /api/v1/machine.
// Usage:
//   - GET /api/v1/machine?search=[MY_MACHINE_LIST|*]
//   - POST /api/v1/machine
//   - PUT /api/v1/machine
//   - DELETE /api/v1/machine?target=[MACHINE_NAME]
// TODO: make logger content qualified.
func (in *Handler) Machine(w http.ResponseWriter, r *http.Request) {
	defer in.logger.Sync()
	defer func() {
		if re := recover(); re != nil {
			in.finalizeError(w, fmt.Errorf("Internal Server Error"), http.StatusInternalServerError)
			in.logger.Error(re)
		}
	}()
	defer in.finalizeHeader(w)

	switch r.Method {
	case "GET":
		// GET implements machine query process.
		targets := strings.Split(r.URL.Query().Get("search"), ",")
		if len(targets) == 1 && targets[0] == "" {
			in.finalizeError(w, fmt.Errorf("Target required"), http.StatusBadRequest)
			in.logger.Errorf("No machine target was specified during query")
			return
		}
		in.logger.Debugf("Search machine: %s", strings.Join(targets, ", "))
		var all bool
		for _, each := range targets {
			if each == "*" {
				all = true
				break
			}
		}
		sortor := genericStorage.NewSortor()
		if all {
			cv := genericStorage.NewMachineList()
			err := in.storage.List(cv)
			if err != nil {
				in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
				in.logger.Error(err)
				return
			}
			for i := range cv.Members {
				sortor.AppendMember(&cv.Members[i])
			}
		} else {
			for _, each := range targets {
				newObj := genericStorage.NewMachine()
				newObj.Name = each
				err := in.storage.Get(newObj)
				if err != nil {
					in.logger.Error(err)
					if !genericStorage.IsInternalError(err) {
						in.finalizeStorageError(w, err)
						return
					}
					in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
					return
				}
				sortor.AppendMember(newObj)
			}
		}
		dAtA, err := json.Marshal(sortor.OrderByName())
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA))
	case "POST":
		// POST implements machine creation process.
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r.Body)
		if err != nil {
			panic(err)
		}
		cv := genericStorage.NewMachine()
		err = json.Unmarshal(buf.Bytes(), cv)
		if err != nil {
			in.finalizeError(w, fmt.Errorf("Invalid Request Body"), http.StatusBadRequest)
			in.logger.Error(err)
			return
		}
		err = in.validateAndFulfillMachine(cv)
		if err != nil {
			in.logger.Error(err)
			in.finalizeError(w, err, http.StatusBadRequest)
			return
		}
		err = in.storage.Create(cv)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		dAtA, err := json.Marshal(cv)
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA), http.StatusCreated)
	case "PUT":
		// PUT implements machine updating process.
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r.Body)
		if err != nil {
			panic(err)
		}
		cv := genericStorage.NewMachine()
		err = json.Unmarshal(buf.Bytes(), cv)
		if err != nil {
			in.finalizeError(w, fmt.Errorf("Invalid Request Body"), http.StatusBadRequest)
			in.logger.Error(err)
			return
		}
		err = in.validateAndFulfillMachine(cv)
		if err != nil {
			in.logger.Error(err)
			in.finalizeError(w, err, http.StatusBadRequest)
			return
		}
		err = in.storage.Update(cv)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		dAtA, err := json.Marshal(cv)
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA))
	case "DELETE":
		// DELETE implements machine deletion process.
		target := r.URL.Query().Get("target")
		if target == "" {
			in.finalizeError(w, fmt.Errorf("Target required"), http.StatusBadRequest)
			in.logger.Errorf("No machine target was specified")
			return
		}
		cv := genericStorage.NewMachine()
		cv.Name = target
		err := in.storage.Delete(cv)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (in *Handler) validateAndFulfillMachine(cv *genericStorage.Machine) error {
	if cv.IPMIUser == "" || cv.IPMIPassword == "" {
		return fmt.Errorf("Invalid IPMI interface credential: %s:%s", cv.IPMIUser, cv.IPMIPassword)
	} else if ip := net.ParseIP(cv.IPMIAddress); ip == nil {
		return fmt.Errorf("Invalid IPMI endpoint: %s", cv.IPMIAddress)
	}
	if cv.SSHPort == 0 {
		cv.SSHPort = 22
	}
	if cv.SSHUser == "" {
		cv.SSHUser = "root"
	}
	return nil
}

// MachineDigest handles requests from /api/v1/machine_digest
// Usage:
//   - GET /api/v1/machine_digest?search=[GUID]
//   - GET /api/v1/machine_digest?scope=machine&target=[MACHINE_NAME]&format=[json]
//   - GET /api/v1/machine_digest?scope=date&target=[DATE]&format=[json|xlsx]
//   - POST /api/v1/machine_digest
// Note that the date must be in unix time format (in seconds).
// TODO: make logger content to be qualified.
func (in *Handler) MachineDigest(w http.ResponseWriter, r *http.Request) {
	defer in.logger.Sync()
	defer func() {
		if re := recover(); re != nil {
			in.finalizeError(w, fmt.Errorf("Internal Server Error"), http.StatusInternalServerError)
			in.logger.Error(re)
		}
	}()
	defer in.finalizeHeader(w)

	if r.Method != "GET" && r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	} else if r.Method == "POST" {
		digest := genericStorage.NewMachineDigest()
		err := in.storage.Create(digest)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}

	search := r.URL.Query().Get("search")
	scope := r.URL.Query().Get("scope")
	rawTarget := r.URL.Query().Get("target")
	format := r.URL.Query().Get("format")

	if search != "" {
		cv := genericStorage.NewMachineDigestList()
		if search == "*" {
			in.logger.Infof("Listing all existing digest...")
			err := in.storage.List(cv, rawTarget)
			if err != nil {
				in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
				in.logger.Error(err)
				return
			}
		} else {
			in.logger.Infof("Search digest with GUID: %s", search)
			obj := genericStorage.NewMachineDigest()
			obj.Name = search
			err := in.storage.Get(obj)
			if err != nil {
				in.logger.Error(err)
				if !genericStorage.IsInternalError(err) {
					in.finalizeStorageError(w, err)
					return
				}
				in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
				return
			}
			cv.Members = append(cv.Members, *obj)
		}
		dAtA, err := json.Marshal(cv.Members)
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA))
		in.logger.Infof("Retrieved digest: %d", len(cv.Members))
		return
	}

	switch format {
	case "":
		format = "json"
	case "json":
	case "xlsx":
	default:
		in.finalizeError(w, fmt.Errorf("Invalid format: %s", format), http.StatusBadRequest)
		return
	}
	switch scope {
	case "machine":
		if format == "xlsx" {
			in.finalizeError(w, fmt.Errorf("XLSX format can only be used with date scope"), http.StatusBadRequest)
			return
		}
		if rawTarget == "" {
			in.finalizeError(w, fmt.Errorf("Target not specified"), http.StatusBadRequest)
			return
		}
		cv := genericStorage.NewMachineSnapshotList()
		err := in.storage.List(cv, rawTarget)
		if err != nil {
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			in.logger.Error(err)
			return
		}
		sortor := genericStorage.NewSortor()
		for i := range cv.Members {
			sortor.AppendMember(&cv.Members[i])
		}
		ordered := sortor.OrderByCreationTimestamp()
		dAtA, err := json.Marshal(ordered)
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA))
	case "date":
		var (
			unixTimestamp int64
			err           error
		)
		if rawTarget != "" {
			unixTimestamp, err = strconv.ParseInt(rawTarget, 10, 64)
			if err != nil {
				in.finalizeError(w, fmt.Errorf("Invalid unix timestamp"), http.StatusBadRequest)
				in.logger.Error(err)
				return
			}
		}
		// If target is not specified, we consider that the user want to acquire a list of current machine info.
		cv := genericStorage.NewMachineDigestList()
		err = in.storage.List(cv)
		if err != nil {
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			in.logger.Error(err)
			return
		}
		sortor := genericStorage.NewSortor()
		for i := range cv.Members {
			// Query by date restricts on completed ones
			if cv.Members[i].State != genericStorage.SuccessState {
				continue
			}
			sortor.AppendMember(&cv.Members[i])
		}
		ordered := sortor.OrderByCreationTimestamp()
		var set genericStorage.SortableObject
		if rawTarget == "" {
			set = ordered[len(ordered)-1]
		} else {
			index := sort.Search(sortor.Len(), func(i int) bool { return ordered[i].GetCreationTimestamp().Unix() > unixTimestamp })
			if index < 1 {
				in.finalizeError(w, fmt.Errorf("Not found"), http.StatusNotFound)
				return
			}
			set = ordered[index-1]
		}
		sortor.Reset()
		for _, each := range set.(*genericStorage.MachineDigest).Members {
			newObj := genericStorage.NewMachineSnapshot()
			newObj.ObjectMeta = each
			err = in.storage.Get(newObj)
			if err != nil {
				in.logger.Error(err)
				if !genericStorage.IsInternalError(err) {
					in.finalizeStorageError(w, err)
					return
				}
				in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
				return
			}
			sortor.AppendMember(newObj)
		}
		ordered = sortor.OrderByNamespace()
		if format == "json" {
			dAtA, err := json.Marshal(ordered)
			if err != nil {
				panic(err)
			}
			in.finalizeJSON(w, bytes.NewReader(dAtA))
			return
		}
		var list []*genericStorage.MachineSnapshot
		for i := range ordered {
			list = append(list, ordered[i].(*genericStorage.MachineSnapshot))
		}
		var buf bytes.Buffer
		err = in.generateXlsxAndWrite(list, &buf)
		if err != nil {
			panic(err)
		}
		in.finalizeXLSX(w, &buf)
		// TODO: support xlsx output
	default:
		in.finalizeError(w, fmt.Errorf("Invalid scope"), http.StatusBadRequest)
	}
}

// DiscoveredMachines handles requests from /api/v1/discovered_machines
// Usage:
//   - GET /api/v1/discovered_machines retrieve a list of unassigned IP addresses (wrapped).
//   - POST /api/v1/discovered_machines rescan immediately.
func (in *Handler) DiscoveredMachines(w http.ResponseWriter, r *http.Request) {
	defer in.logger.Sync()
	defer func() {
		if re := recover(); re != nil {
			in.finalizeError(w, fmt.Errorf("Internal Server Error"), http.StatusInternalServerError)
			in.logger.Error(re)
		}
	}()
	defer in.finalizeHeader(w)

	switch r.Method {
	case "GET":
		latest := genericStorage.NewDiscoveredMachines()
		err := in.storage.Get(latest)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		dAtA, err := json.Marshal(latest)
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA), http.StatusOK)
	case "POST":
		latest := genericStorage.NewDiscoveredMachines()
		err := in.storage.Get(latest)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		if latest.State == genericStorage.SuccessState || latest.State == genericStorage.FailureState {
			latest.State = genericStorage.StartedState
		} else {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		err = in.storage.Update(latest)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// AutoDiscovery handles requests from /api/v1/auto_discovery
// Usage:
//   - GET /api/v1/auto_discovery?search=[*|SUBNET_LIST]
//   - POST /api/v1/auto_discovery
//   - PUT /api/v1/auto_discovery
//   - DELETE /api/v1/auto_discovery?target=[SUBNET_NAME]
func (in *Handler) AutoDiscovery(w http.ResponseWriter, r *http.Request) {
	defer in.logger.Sync()
	defer func() {
		if re := recover(); re != nil {
			in.finalizeError(w, fmt.Errorf("Internal Server Error"), http.StatusInternalServerError)
			in.logger.Error(re)
		}
	}()
	defer in.finalizeHeader(w)

	switch r.Method {
	case "GET":
		targets := strings.Split(r.URL.Query().Get("search"), ",")
		if len(targets) == 1 && targets[0] == "" {
			in.finalizeError(w, fmt.Errorf("Target required"), http.StatusBadRequest)
			in.logger.Errorf("No auto discovery target was specified during query")
			return
		}
		in.logger.Debugf("Search auto discovery: %s", strings.Join(targets, ", "))
		var all bool
		for _, each := range targets {
			if each == "*" {
				all = true
				break
			}
		}
		sortor := genericStorage.NewSortor()
		if all {
			cv := genericStorage.NewAutoDiscoveryList()
			err := in.storage.List(cv)
			if err != nil {
				in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
				in.logger.Error(err)
				return
			}
			for i := range cv.Members {
				sortor.AppendMember(&cv.Members[i])
			}
		} else {
			for _, each := range targets {
				newObj := genericStorage.NewMachine()
				newObj.Name = each
				err := in.storage.Get(newObj)
				if err != nil {
					in.logger.Error(err)
					if !genericStorage.IsInternalError(err) {
						in.finalizeStorageError(w, err)
						return
					}
					in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
					return
				}
				sortor.AppendMember(newObj)
			}
		}
		dAtA, err := json.Marshal(sortor.OrderByName())
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA))
	case "POST":
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r.Body)
		if err != nil {
			panic(err)
		}
		cv := genericStorage.NewAutoDiscovery()
		err = json.Unmarshal(buf.Bytes(), cv)
		if err != nil {
			in.finalizeError(w, fmt.Errorf("Invalid Request Body"), http.StatusBadRequest)
			in.logger.Error(err)
			return
		}
		_, _, err = net.ParseCIDR(cv.CIDR)
		if err != nil {
			in.logger.Error(err)
			in.finalizeError(w, err, http.StatusBadRequest)
			return
		}
		err = in.storage.Create(cv)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		dAtA, err := json.Marshal(cv)
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA), http.StatusCreated)
	case "PUT":
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r.Body)
		if err != nil {
			panic(err)
		}
		cv := genericStorage.NewAutoDiscovery()
		err = json.Unmarshal(buf.Bytes(), cv)
		if err != nil {
			in.finalizeError(w, fmt.Errorf("Invalid Request Body"), http.StatusBadRequest)
			in.logger.Error(err)
			return
		}
		_, _, err = net.ParseCIDR(cv.CIDR)
		if err != nil {
			in.logger.Error(err)
			in.finalizeError(w, err, http.StatusBadRequest)
			return
		}
		err = in.storage.Update(cv)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		dAtA, err := json.Marshal(cv)
		if err != nil {
			panic(err)
		}
		in.finalizeJSON(w, bytes.NewReader(dAtA))
	case "DELETE":
		target := r.URL.Query().Get("target")
		if target == "" {
			in.finalizeError(w, fmt.Errorf("Target required"), http.StatusBadRequest)
			in.logger.Errorf("No machine target was specified")
			return
		}
		cv := genericStorage.NewAutoDiscovery()
		cv.Name = target
		err := in.storage.Delete(cv)
		if err != nil {
			in.logger.Error(err)
			if !genericStorage.IsInternalError(err) {
				in.finalizeStorageError(w, err)
				return
			}
			in.finalizeError(w, fmt.Errorf("Database Failure"), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

const (
	// DEF_XLSX_SHEET is the name of default xlsx sheet
	DEF_XLSX_SHEET = "Overview"
)

func (in *Handler) generateXlsxAndWrite(list []*genericStorage.MachineSnapshot, writer io.Writer) error {
	buf := excel.NewFile()
	buf.SetSheetName(buf.GetSheetName(1), DEF_XLSX_SHEET)
	/* ------------ WRITE TOP HEADER ------------ */
	buf.MergeCell(DEF_XLSX_SHEET, "A1", "F1")
	buf.SetCellStr(DEF_XLSX_SHEET, "A1", "Basic")
	buf.MergeCell(DEF_XLSX_SHEET, "G1", "I1")
	buf.SetCellStr(DEF_XLSX_SHEET, "G1", "Specification")
	buf.MergeCell(DEF_XLSX_SHEET, "J1", "K1")
	buf.SetCellStr(DEF_XLSX_SHEET, "J1", "Location")
	buf.MergeCell(DEF_XLSX_SHEET, "L1", "O1")
	buf.SetCellStr(DEF_XLSX_SHEET, "L1", "CPU")
	buf.MergeCell(DEF_XLSX_SHEET, "P1", "Q1")
	buf.SetCellStr(DEF_XLSX_SHEET, "P1", "Memory")
	buf.MergeCell(DEF_XLSX_SHEET, "R1", "S1")
	buf.SetCellStr(DEF_XLSX_SHEET, "R1", "Disk")
	buf.MergeCell(DEF_XLSX_SHEET, "T1", "X1")
	buf.SetCellStr(DEF_XLSX_SHEET, "T1", "Network")
	/* ------------ WRITE PRIMARY HEADER ------------ */
	buf.MergeCell(DEF_XLSX_SHEET, "A2", "A3")
	buf.SetCellStr(DEF_XLSX_SHEET, "A2", "No.")
	buf.MergeCell(DEF_XLSX_SHEET, "B2", "B3")
	buf.SetCellStr(DEF_XLSX_SHEET, "B2", "Name")
	buf.MergeCell(DEF_XLSX_SHEET, "C2", "C3")
	buf.SetCellStr(DEF_XLSX_SHEET, "C2", "OS")
	buf.MergeCell(DEF_XLSX_SHEET, "D2", "D3")
	buf.SetCellStr(DEF_XLSX_SHEET, "D2", "Department")
	buf.MergeCell(DEF_XLSX_SHEET, "E2", "E3")
	buf.SetCellStr(DEF_XLSX_SHEET, "E2", "Type")
	buf.MergeCell(DEF_XLSX_SHEET, "F2", "F3")
	buf.SetCellStr(DEF_XLSX_SHEET, "F2", "Comment")
	buf.MergeCell(DEF_XLSX_SHEET, "G2", "G3")
	buf.SetCellStr(DEF_XLSX_SHEET, "G2", "Manufacturer")
	buf.MergeCell(DEF_XLSX_SHEET, "H2", "H3")
	buf.SetCellStr(DEF_XLSX_SHEET, "H2", "Model")
	buf.MergeCell(DEF_XLSX_SHEET, "I2", "I3")
	buf.SetCellStr(DEF_XLSX_SHEET, "I2", "Serial Num")
	buf.MergeCell(DEF_XLSX_SHEET, "J2", "J3")
	buf.SetCellStr(DEF_XLSX_SHEET, "J2", "Rack Name")
	buf.MergeCell(DEF_XLSX_SHEET, "K2", "K3")
	buf.SetCellStr(DEF_XLSX_SHEET, "K2", "Rack Slot")
	buf.MergeCell(DEF_XLSX_SHEET, "L2", "L3")
	buf.SetCellStr(DEF_XLSX_SHEET, "L2", "Model")
	buf.MergeCell(DEF_XLSX_SHEET, "M2", "M3")
	buf.SetCellStr(DEF_XLSX_SHEET, "M2", "Base Freq")
	buf.MergeCell(DEF_XLSX_SHEET, "N2", "N3")
	buf.SetCellStr(DEF_XLSX_SHEET, "N2", "Count")
	buf.MergeCell(DEF_XLSX_SHEET, "O2", "O3")
	buf.SetCellStr(DEF_XLSX_SHEET, "O2", "Cores")
	buf.MergeCell(DEF_XLSX_SHEET, "P2", "P3")
	buf.SetCellStr(DEF_XLSX_SHEET, "P2", "DIMMs")
	buf.MergeCell(DEF_XLSX_SHEET, "Q2", "Q3")
	buf.SetCellStr(DEF_XLSX_SHEET, "Q2", "Capacity")
	buf.MergeCell(DEF_XLSX_SHEET, "R2", "S2")
	buf.SetCellStr(DEF_XLSX_SHEET, "R2", "Virtual Disks")
	buf.SetCellStr(DEF_XLSX_SHEET, "R3", "Name")
	buf.SetCellStr(DEF_XLSX_SHEET, "S3", "Size")
	buf.MergeCell(DEF_XLSX_SHEET, "T2", "T3")
	buf.SetCellStr(DEF_XLSX_SHEET, "T2", "Primary IP")
	buf.MergeCell(DEF_XLSX_SHEET, "U2", "U3")
	buf.SetCellStr(DEF_XLSX_SHEET, "U2", "IPMI Address")
	buf.MergeCell(DEF_XLSX_SHEET, "V2", "X2")
	buf.SetCellStr(DEF_XLSX_SHEET, "V2", "Logical Interfaces")
	buf.SetCellStr(DEF_XLSX_SHEET, "V3", "Name")
	buf.SetCellStr(DEF_XLSX_SHEET, "W3", "Member")
	buf.SetCellStr(DEF_XLSX_SHEET, "X3", "MAC")
	/* ------------ WRITE ROW ------------ */
	row := 4
	axisFactory := func(col rune, r int) string {
		return fmt.Sprintf("%s%d", string(col), r)
	}
	for num, qr := range list {
		lineHeight := 1
		buf.SetCellInt(DEF_XLSX_SHEET, axisFactory('A', row), num+1)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('B', row), qr.Namespace) // use namespace as its name
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('C', row), qr.OS)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('D', row), qr.Department)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('E', row), qr.Type)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('F', row), qr.Comment)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('G', row), qr.Manufacturer)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('H', row), qr.Model)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('I', row), qr.SerialNumber)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('J', row), qr.Location.RackName)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('K', row), qr.Location.RackSlot)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('L', row), qr.CPU.Model)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('M', row), qr.CPU.BaseFreq)
		buf.SetCellValue(DEF_XLSX_SHEET, axisFactory('N', row), qr.CPU.Count)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('O', row), fmt.Sprintf("%d / %d", qr.CPU.Cores, qr.CPU.Threads))
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('P', row), fmt.Sprintf("%d / %d", qr.Memory.PopulatedDIMMs, qr.Memory.MaximumDIMMs))
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('Q', row), qr.Memory.InstalledMemory)
		if h := len(qr.Storage.VirtualDisks); h > lineHeight {
			lineHeight = h
		}
		for vdi := range qr.Storage.VirtualDisks {
			buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('R', row+vdi), qr.Storage.VirtualDisks[vdi].Name)
			buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('S', row+vdi), qr.Storage.VirtualDisks[vdi].Size)
		}
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('T', row), qr.Network.PrimaryIPAddress)
		buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('U', row), qr.Network.IPMIAddress)
		var h int
		for lii := range qr.Network.LogicalInterfaces {
			length := len(qr.Network.LogicalInterfaces[lii].Members)
			buf.MergeCell(DEF_XLSX_SHEET, axisFactory('V', row+h), axisFactory('V', row+h+length-1))
			buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('V', row+h), qr.Network.LogicalInterfaces[lii].Name)
			for limi := range qr.Network.LogicalInterfaces[lii].Members {
				buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('W', row+h+limi), qr.Network.LogicalInterfaces[lii].Members[limi].Name)
				buf.SetCellStr(DEF_XLSX_SHEET, axisFactory('X', row+h+limi), qr.Network.LogicalInterfaces[lii].Members[limi].MACAddress)
			}
			h += length
		}
		if h > lineHeight {
			lineHeight = h
		}
		if lineHeight > 1 {
			cols2align := "ABCDEFGHIJKLMNOPQTU"
			for _, each := range cols2align {
				buf.MergeCell(DEF_XLSX_SHEET, axisFactory(each, row), axisFactory(each, row+lineHeight-1))
			}
		}
		row += lineHeight
	}
	return buf.Write(writer)
}

func (in *Handler) finalizeStorageError(w http.ResponseWriter, e error) {
	if genericStorage.IsNotFound(e) {
		in.finalizeError(w, e, http.StatusNotFound)
	}
	if genericStorage.IsConflict(e) {
		in.finalizeError(w, e, http.StatusConflict)
	}
}

func (in *Handler) finalizeBody(w http.ResponseWriter, r io.Reader, status int) {
	w.WriteHeader(status)
	length, err := io.Copy(w, r)
	if err != nil {
		// Panic here due to there is no possible way to recover.
		panic(err)
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
}

func (in *Handler) finalizeJSON(w http.ResponseWriter, r io.Reader, status ...int) {
	var stat int
	if len(status) > 0 {
		stat = status[0]
	} else {
		stat = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	in.finalizeBody(w, r, stat)
}

func (in *Handler) finalizeXLSX(w http.ResponseWriter, r io.Reader) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	in.finalizeBody(w, r, http.StatusOK)
}

func (in *Handler) finalizeError(w http.ResponseWriter, e error, status int) {
	in.finalizeBody(w, bytes.NewReader([]byte(e.Error()+"\n")), status)
	w.Header().Set("Content-Type", "plain/text; charset=utf-8")
}

func (in *Handler) finalizeNull(w http.ResponseWriter) {
	defer in.finalizeHeader(w)
	w.WriteHeader(http.StatusNoContent)
}

func (in *Handler) finalizeHeader(w http.ResponseWriter) {
	w.Header().Set("Date", time.Now().Format("Mon, 2 Jan 2006 15:04:05 MST"))
	w.Header().Set("Server", "CMDB/v0.1-alpha")
}

func (in *Handler) auditIntercepter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer in.logger.Sync()
		// The incoming request might be forwarded from a proxy server.
		ips := r.Header.Get("X-Forwarded-For")
		if ips == "" {
			ips = r.RemoteAddr
		}
		ip := strings.TrimSpace(strings.Split(ips, ",")[0])
		if ip == "" {
			ip = "local"
		}
		in.logger.Infow("Request =>",
			"user_agent", r.Header.Get("User-Agent"),
			"method", r.Method,
			"url", r.URL.Path,
			"from", ip,
		)
		next.ServeHTTP(w, r)
	})
}

// NewHandler returns a new initialized HTTP handler
func NewHandler(storage genericStorage.Storage, logger *zap.SugaredLogger, wwwroot string) *Handler {
	root := mux.NewRouter()
	h := &Handler{
		Router:  root,
		wwwroot: wwwroot,
		storage: storage,
		logger:  logger,
	}
	root.Use(h.auditIntercepter)

	apiRoot := root.PathPrefix(apis.APIv1).Subrouter()
	apiRoot.HandleFunc(apis.MachineAPI, h.Machine)
	apiRoot.HandleFunc(apis.MachineDigestAPI, h.MachineDigest)
	apiRoot.HandleFunc(apis.DiscoveredMachinesAPI, h.DiscoveredMachines)
	apiRoot.HandleFunc(apis.AutoDiscoveryAPI, h.AutoDiscovery)

	root.PathPrefix("/").HandlerFunc(h.Frontend)
	return h
}
