// Copyright 2018 Alfred Chou <unioverlord@gmail.com>
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

package net

import (
	"fmt"
	"net"
	"time"
)

var (
	// DefaultTCPTimeout is the default TCP timeout threshold.
	DefaultTCPTimeout = 3 * time.Second
)

// ScanCIDR scans TCP ports on given cidr, issue a list of scanning result and returns
// any encountered error.
func ScanCIDR(cidr string, port ...int) (result map[string]map[int]bool, err error) {
	var (
		ip    net.IP
		ipNet *net.IPNet
	)
	ip, ipNet, err = net.ParseCIDR(cidr)
	if err != nil {
		return
	}
	// Calc available IPs from given CIDR. The first IP (network address) and the last
	// (broadcast address) should be dropped.
	var ips []string
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); {
		ips = append(ips, ip.String())
		for i := len(ip) - 1; i >= 0; i-- {
			ip[i]++
			if ip[i] > 0 {
				break
			}
		}
	}
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	} else {
		// None avaiable to use except the network address and the broadcast address.
		ips = ips[:0]
	}
	out := make(chan map[string]map[int]bool, 1)
	defer close(out)
	scan := func(ips []string, out chan<- map[string]map[int]bool) {
		scanned := make(map[string]map[int]bool)
		for _, each := range ips {
			r, e := ScanIP(each, port...)
			if e != nil {
				continue
			}
			scanned[each] = r
		}
		out <- scanned
	}
	go scan(ips[:len(ips)/2], out)
	go scan(ips[len(ips)/2:], out)
	t1, t2 := <-out, <-out
	result = t1
	for k, v := range t2 {
		result[k] = v
	}
	return
}

// ScanIP scans on given IP's given port, returns the scanned status and any encountered error.
func ScanIP(ip string, port ...int) (result map[int]bool, err error) {
	check := net.ParseIP(ip)
	if check == nil {
		// Generate a underlay error if invalid.
		return nil, &net.ParseError{Type: "IP address", Text: ip}
	}
	out := make(chan []int, 1)
	defer close(out)
	scan := func(ports []int, out chan<- []int) {
		var opened []int
		for _, each := range ports {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, each), DefaultTCPTimeout)
			if err != nil {
				continue
			}
			conn.Close()
			opened = append(opened, each)
		}
		out <- opened
	}
	go scan(port[:len(port)/2], out)
	go scan(port[len(port)/2:], out)
	t1, t2 := <-out, <-out
	result = make(map[int]bool)
	for _, each := range append(t1, t2...) {
		result[each] = true
	}
	for _, each := range port {
		if _, ok := result[each]; !ok {
			result[each] = false
		}
	}
	return
}
