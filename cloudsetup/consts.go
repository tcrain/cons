/*
github.com/tcrain/cons - Experimental project for testing and scaling consensus algorithms.
Copyright (C) 2020 The project authors - tcrain

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/

package cloudsetup

import (
	"github.com/tcrain/cons/config"
	"google.golang.org/api/compute/v1"
	"os"
	"time"
)

var JsonPath = os.Getenv("OAUTHPATH")
var ProjectID = os.Getenv("PROJECTID")

const (
	AllowPreemption     = false
	DefaultInstanceType = "n1-standard-2"
	TemplateName        = "base-cons-template"

	DefaultRegion     = "us-central1" // "us-east1"
	BaseInstanceName  = "cons-instance"
	InstanceGroupName = "cons-instance-group"
	DefaultZone       = "us-central1-a"
	InitInstance      = "initial-instance"
	Port              = config.RPCNodePort
	FirewallName      = "cons-firewall"
)

var retryTime = 8000 * time.Millisecond

const (
	GlobalImageName = "projects/debian-cloud/global/images/family/debian-10"
	// GlobalImageName   = "projects/centos-cloud/global/images/family/centos-8"
	// initScriptName = "./scripts/imagesetuplocal.sh"
	ConsImageName = "cons-image"
	// goVersion = "1.14.2"
)

var consFirewall = &compute.Firewall{
	Allowed:      []*compute.FirewallAllowed{{IPProtocol: "tcp"}, {IPProtocol: "udp"}},
	Description:  "Open firewall for experiments",
	Direction:    "INGRESS",
	Name:         FirewallName,
	SourceRanges: []string{"0.0.0.0/0"}, // TODO only allow specific ports
}
