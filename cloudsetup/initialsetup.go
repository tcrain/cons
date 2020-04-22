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
	"fmt"
	"github.com/tcrain/cons/consensus/logging"
	"google.golang.org/api/compute/v1"
	"strings"
	"time"
)

func getMyTemplateURL(project string, templateName string) string {
	return fmt.Sprintf("projects/%v/global/instanceTemplates/%v", project, templateName)
}

func getMyImageURL(project, imageName string) string {
	return fmt.Sprintf("projects/%v/global/images/%v", project, imageName)
}

// UndoInitialSetup removes the items created in the initial setup
func UndoInitialSetup(project, zone string, service *compute.Service) (err error) {

	// Delete the image
	if err = DeleteImage(ConsImageName, project, zone, service); err != nil {
		logging.Errorf("error deleting image %v:", ConsImageName, err)
	}

	return DeleteInstanceTemplate(TemplateName, project, service)
}

// DoDeleteDisk deletes the given disk.
func DoDeleteDisk(instanceName, project, zone string, service *compute.Service) (err error) {
	logging.Printf("Deleting disk from instance %v, project %v, zone %v", instanceName, project, zone)
	if _, err := service.Disks.Delete(project, zone, instanceName).Do(); err != nil {
		return err
	}

	for true {
		if dsk, err := service.Disks.Get(project, zone, instanceName).Do(); err == nil {
			logging.Printf("Waiting for disk to be deleted, status %v", dsk.Status)
			time.Sleep(retryTime)
			continue
		}
		break
	}
	return nil
}

// DoLaunchInstance launches an instance.
func DoShutdownInstance(instanceName, project, zone string, service *compute.Service) (err error) {

	return ShutdownInstance(instanceName, project, zone, service)
}

// DoAddFirewallRule adds the cons firewall rule.
func DoAllFirewallRule(project string, service *compute.Service) (err error) {
	if _, err = service.Firewalls.Insert(project, consFirewall).Do(); err != nil {
		return
	}
	return
}

// DoRemoveFirewallRule removes the cons firewall rule.
func DoRemoveFirewallRule(project string, service *compute.Service) (err error) {
	if _, err = service.Firewalls.Delete(project, FirewallName).Do(); err != nil {
		return
	}
	return
}

// DoLaunchInstance launches an instance.
func DoLaunchInstance(instanceName, imageName, instanceType, project, zone string, service *compute.Service) (err error) {
	// First launch an image

	//check if it is a local image
	if strings.Count(imageName, "/") == 0 {
		imageName = getMyImageURL(project, imageName)
	}

	var inst *compute.Instance
	if inst, err = LaunchCreateNode(instanceName, imageName, instanceType, project, zone, service); err != nil {
		logging.Error(err)
		return
	}

	/*	defer func () {
			if err := ShutdownInstance(initInstance, project, zone, service); err != nil {
				logging.Print("error shutting down initial instance: ", err)
			}
		} ()
	*/
	fmt.Println(GetIPFromInstance(inst))
	return nil
}

// DoGetInstanceIP gets the IP on an instance launched with DoLaunchInstance.
func DoGetInstanceIP(instanceName, imageName, instanceType, project, zone string, service *compute.Service) (err error) {
	// First launch an image

	//check if it is a local image
	if strings.Count(imageName, "/") == 0 {
		imageName = getMyImageURL(project, imageName)
	}

	var inst *compute.Instance
	if inst, err = GetInstance(instanceName, imageName, instanceType, project, zone, service); err != nil {
		logging.Error(err)
		return
	}

	fmt.Println(GetIPFromInstance(inst))
	return nil
}

// DoInitialSetup creates an image and image template from the values.
func DoInitialSetup(instanceName, instanceType, project, zone string, service *compute.Service) (err error) {

	/*	// Now run the initial setup script
		if err = SetupInitialNode(GetIPFromInstance(inst), userName, key, goVersion, branch, false); err != nil {
			logging.Error(err)
			return
		}*/

	// Now make an image
	if err = MakeImageFromInstance(ConsImageName, instanceName, project, zone, service); err != nil {
		logging.Error(err)
		return
	}

	// Now make an instance template
	if err = CreateInstanceTemplate(TemplateName, GlobalImageName, instanceType, project, service); err != nil {
		logging.Error(err)
		return
	}

	return nil
}

func DeleteImage(imageName, project, zone string, service *compute.Service) error {

	logging.Printf("Calling delete on image %v, project %v, zone %v", imageName, project, zone)
	_, err := service.Images.Delete(project, imageName).Do()

	for true {
		if img, err := service.Images.Get(project, imageName).Do(); err == nil {
			logging.Printf("Waiting for image deletion, status: %v", img.Status)
			time.Sleep(retryTime)
			continue
		}
		break
	}
	return err
}

func CreateInstanceTemplate(templateName, imageName, instanceType, project string, service *compute.Service) (err error) {

	logging.Printf("Creating instance template %v, from image %v, instance type %v, project %v",
		templateName, imageName, instanceType, project)
	template := &compute.InstanceTemplate{
		Name: templateName,
		Properties: &compute.InstanceProperties{
			Scheduling: &compute.Scheduling{
				Preemptible: AllowPreemption,
			},
			MachineType: instanceType,
			NetworkInterfaces: []*compute.NetworkInterface{
				{Network: "global/networks/default",
					AccessConfigs: []*compute.AccessConfig{
						{Name: "external-IP",
							Type: "ONE_TO_ONE_NAT",
						},
					},
				},
			},
			Disks: []*compute.AttachedDisk{
				{Type: "PERSISTENT",
					Boot: true,
					Mode: "READ_WRITE",
					InitializeParams: &compute.AttachedDiskInitializeParams{
						SourceImage: imageName, //getMyImageURL(project, imageName),
					},
				},
			},
		},
	}

	_, err = service.InstanceTemplates.Insert(project, template).Do()
	if err != nil {
		logging.Error(err)
		return
	}

	if err = waitInstanceTemplateReady(templateName, project, service); err != nil {
		return err
	}

	return nil
}

func waitInstanceTemplateReady(templateName, project string, service *compute.Service) (err error) {
	// wait until the template is ready
	for true {
		if _, err = service.InstanceTemplates.Get(project, templateName).Do(); err != nil {
			logging.Print("instance template not yet ready, will retry after sleep")
			time.Sleep(retryTime)
			continue
		}
		break
	}

	return nil
}

func GetIPFromInstance(inst *compute.Instance) string {

	return inst.NetworkInterfaces[0].AccessConfigs[0].NatIP
}

func ShutdownInstance(instanceName, project, zone string, service *compute.Service) error {

	logging.Printf("Calling delete on instance %v, project %v, zone %v", instanceName, project, zone)
	if _, err := service.Instances.Delete(project, zone, instanceName).Do(); err != nil {
		return err
	}
	for true {
		if inst, err := service.Instances.Get(project, zone, instanceName).Do(); err == nil {
			logging.Printf("Waiting for instance to shut down status: %v", inst.Status)
			time.Sleep(retryTime)
			continue
		}
		break
	}
	return nil
}

func MakeImageFromInstance(imageName, instanceName, project, zone string, service *compute.Service) error {
	logging.Printf("Making image %v from instance %v, zone %v, project %v", imageName, instanceName,
		project, zone)

	// First shutdown the image
	// if err := ShutdownInstance(instanceName, project, zone, service); err != nil {
	//	logging.Error(err)
	// return err
	//}

	/*	retIst, err := service.Instances.Get(project, zone, instanceName).Do()
		if err != nil {
			logging.Error(err)
			return err
		}
		sourceDisk := retIst.Disks[0].DeviceName*/

	img := &compute.Image{
		Name:       imageName,
		SourceDisk: fmt.Sprintf("/zones/%v/disks/%v", zone, instanceName),
	}

	if _, err := service.Images.Insert(project, img).Do(); err != nil {
		logging.Error(err)
		return err
	}

	// Wait until the image is ready
	for true {
		img, err := service.Images.Get(project, imageName).Do()
		if err != nil {
			logging.Print("error getting created image, will retry after sleep")
			time.Sleep(retryTime)
			continue
		}
		switch img.Status {
		case "DELETING":
			logging.Error(err)
			return fmt.Errorf("image was deleted")
		case "FAILED":
			logging.Error(err)
			return fmt.Errorf("image creation failed")
		case "PENDING":
			logging.Print("image not yet ready, will retry after sleep")
			time.Sleep(retryTime)
			continue
		case "READY": // ok
		}
		break
	}
	logging.Print("done creation of image ", imageName)

	return nil
}

func GetInstance(name, imageName, instanceType, project, zone string, service *compute.Service) (*compute.Instance, error) {
	retIst, err := service.Instances.Get(project, zone, name).Do()
	if err != nil {
		logging.Error(err)
		return nil, err
	}
	logging.Print("Found instance ", name)
	return retIst, nil
}

func LaunchCreateNode(name, imageName, instanceType, project, zone string, service *compute.Service) (*compute.Instance, error) {

	logging.Printf("Launching instance name %v, image %v, type %v, project %v, zone %v",
		name, imageName, instanceType, project, zone)
	inst := &compute.Instance{
		Name: name,
		Scheduling: &compute.Scheduling{
			Preemptible: false,
		},
		MachineType: fmt.Sprintf("/zones/%v/machineTypes/%v", zone, instanceType),
		NetworkInterfaces: []*compute.NetworkInterface{
			{Network: "global/networks/default",
				AccessConfigs: []*compute.AccessConfig{
					{Name: "external-IP",
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Disks: []*compute.AttachedDisk{
			{Type: "PERSISTENT",
				Boot: true,
				Mode: "READ_WRITE",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: imageName,
				},
			},
		},
	}

	_, err := service.Instances.Insert(project, zone, inst).Do()
	if err != nil {
		logging.Error(err)
		return nil, err
	}

	// wait for the instance to be ready
	var retIst *compute.Instance
	for true {
		retIst, err = service.Instances.Get(project, zone, name).Do()
		if err != nil {
			logging.Error(err)
			return nil, err
		}
		if retIst.Status == "RUNNING" {
			break
		}
		logging.Printf("Instance %v status %v, waiting.", name, retIst.Status)
		time.Sleep(retryTime)
	}
	logging.Print("successfully launched instance ", name)
	return retIst, nil
}
