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
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/tcrain/cons/cloudsetup"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"log"
	"strings"
)

func main() {

	var project string
	var credentialPath string
	var instanceType string
	var imageName string

	var doLaunchInstance bool
	var doShutdownInstance bool
	var instanceName string
	var doDeleteDisk bool

	var doCreateImage bool
	var initialSetupZone string

	var doRebootRegions bool

	var doLaunchRegions bool
	var singleZoneRegion bool
	var notSingleZoneRegion bool
	var launchRegions string
	var nodeCount int

	var undoLaunchRegions bool
	var doGetIPs bool

	var doGetInstanceIP bool

	var undoCreateImage bool

	var doCreateInstanceTemplate bool
	var doDeleteInstanceTemplate bool

	flag.BoolVar(&doCreateInstanceTemplate, "cit", false, "Create instance template,"+
		"use with options: p, i")

	flag.BoolVar(&doDeleteInstanceTemplate, "dit", false, "Remove instance template,"+
		"use with options: p")

	flag.StringVar(&project, "p", cloudsetup.ProjectID, "Cloud project ID")
	flag.StringVar(&credentialPath, "c", cloudsetup.JsonPath, "Path to credentials file")
	flag.StringVar(&instanceType, "i", cloudsetup.DefaultInstanceType, "Instance type to use")
	flag.StringVar(&instanceName, "in", cloudsetup.InitInstance, "Init instance name")
	flag.StringVar(&imageName, "im", cloudsetup.GlobalImageName, "Image name to use when launching a node")

	flag.BoolVar(&doShutdownInstance, "sd", false, "Shutdown an instance,"+
		"use this with options: "+
		"im, p, z")
	flag.BoolVar(&doDeleteDisk, "dd", false, "Delete a disk, "+
		"use this with options: im, p, z")

	flag.BoolVar(&doLaunchInstance, "li", false, "Launch an instance,"+
		"the output of this command is the ip of the image, use this with options: "+
		"in, i, im, p, z")

	flag.BoolVar(&doGetInstanceIP, "ii", false, "Get the ip of an instance launched with -li,"+
		"the output of this command is the ip of the image, use this with options: "+
		"in, i, im, p, z")

	flag.BoolVar(&doCreateImage, "s", false, "This should be called after an instance is setup,"+
		"this will use the instance to create an image, use this with options:"+
		"i, im, in, p")
	flag.StringVar(&initialSetupZone, "z", cloudsetup.DefaultZone, "Zone to perform initial setup in")
	flag.BoolVar(&undoCreateImage, "us", false, "Undo create image setup, use this with options:"+
		"in, p")

	flag.BoolVar(&doLaunchRegions, "lr", false, "Launch instance groups in multiple zones, use this with options:"+
		"n, p, r")
	flag.BoolVar(&singleZoneRegion, "lrz", false, "For instance group commands use a single zone:"+
		"n, p, r")
	flag.BoolVar(&doRebootRegions, "rb", false, "Reboot instance groups in multiple zones, use this with options:"+
		"n, p, r")

	flag.BoolVar(&notSingleZoneRegion, "nlrz", false, "Unused")

	flag.StringVar(&launchRegions, "r", cloudsetup.DefaultRegion, "Regions to launch instance groups on")
	flag.IntVar(&nodeCount, "n", 1, "Nodes per region to launch")
	flag.BoolVar(&undoLaunchRegions, "ur", false, "Undo region launch, use this with options:"+
		"p, r")
	flag.BoolVar(&doGetIPs, "gi", false, "Get the IPs for the running instance groups,"+
		"use with options: p, r")

	flag.Parse()

	ctx := context.Background()
	service, err := compute.NewService(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		log.Fatal(err)
	}

	if doLaunchInstance {
		if err := cloudsetup.DoLaunchInstance(instanceName, imageName, instanceType, project,
			initialSetupZone, service); err != nil {
			log.Fatal(err)
		}
	}

	if doGetInstanceIP {
		if err := cloudsetup.DoGetInstanceIP(instanceName, imageName, instanceType, project,
			initialSetupZone, service); err != nil {
			log.Fatal(err)
		}
	}

	if doShutdownInstance {
		if err := cloudsetup.DoShutdownInstance(instanceName, project, initialSetupZone, service); err != nil {
			log.Fatal(err)
		}
	}

	if doDeleteDisk {
		if err := cloudsetup.DoDeleteDisk(instanceName, project, initialSetupZone, service); err != nil {
			log.Fatal(err)
		}
	}

	if doCreateImage {
		if err := cloudsetup.MakeImageFromInstance(imageName, instanceName, project, initialSetupZone,
			service); err != nil {
			log.Fatal(err)
		}
	}

	if undoCreateImage {
		if err := cloudsetup.DeleteImage(imageName, project, initialSetupZone, service); err != nil {
			log.Fatal(err)
		}
	}

	if doCreateInstanceTemplate {
		if err := cloudsetup.CreateInstanceTemplate(cloudsetup.TemplateName, cloudsetup.GlobalImageName,
			instanceType, project, service); err != nil {
			log.Fatal(err)
		}
	}

	if doDeleteInstanceTemplate {
		if err := cloudsetup.DeleteInstanceTemplate(cloudsetup.TemplateName, project, service); err != nil {
			log.Fatal(err)
		}
	}

	regions := strings.Fields(launchRegions)
	if doGetIPs {
		ips, err := cloudsetup.GetInstanceGroupsIPs(singleZoneRegion, cloudsetup.InstanceGroupName, project, regions, service)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(strings.Join(ips, "\n"))
	}

	if doLaunchRegions {
		if err := cloudsetup.LaunchInstanceGroups(singleZoneRegion, nodeCount, cloudsetup.BaseInstanceName,
			cloudsetup.InstanceGroupName, cloudsetup.TemplateName,
			project, regions, service); err != nil {

			log.Fatal(err)
		}
	}

	if undoLaunchRegions {
		if errs := cloudsetup.ShutdownInstanceGroups(singleZoneRegion, cloudsetup.InstanceGroupName,
			project, regions, service); len(errs) > 0 {

			log.Fatal(errs)
		}
	}

	if doRebootRegions {
		if errs := cloudsetup.DoRebootRegions(singleZoneRegion, cloudsetup.InstanceGroupName,
			project, regions, service); len(errs) > 0 {

			log.Fatal(errs)
		}
	}

}
