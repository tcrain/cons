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
	"sync"
	"time"
)

func getZoneAndName(instanceURL string) (zone, name string) {
	name = instanceURL[strings.LastIndex(instanceURL, "/")+1:]
	zone = strings.Split(instanceURL, "/")[8]
	return
}

func ShutdownInstanceGroups(singleZone bool, instanceGroupName, project string, regions []string, service *compute.Service) (ret []error) {
	logging.Print("Removing firewall rule")
	if err := DoRemoveFirewallRule(project, service); err != nil {
		ret = append(ret, err)
	}

	// First get their disks
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var igmList [][]*compute.ManagedInstance
	for _, region := range regions {
		if singleZone {
			igm, err := service.InstanceGroupManagers.ListManagedInstances(project, region, instanceGroupName).Do()
			if err != nil {
				logging.Error(err)
				ret = append(ret, err)
				continue
			}
			igmList = append(igmList, igm.ManagedInstances)

		} else {
			igm, err := service.RegionInstanceGroupManagers.ListManagedInstances(project, region, instanceGroupName).Do()
			if err != nil {
				logging.Error(err)
				ret = append(ret, err)
				continue
			}
			igmList = append(igmList, igm.ManagedInstances)
		}
	}

	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			logging.Printf("shutting down instance group %v, region %v, project %v", instanceGroupName, region, project)
			var err error
			if singleZone {
				_, err = service.InstanceGroupManagers.Delete(project, region, instanceGroupName).Do()
			} else {
				_, err = service.RegionInstanceGroupManagers.Delete(project, region, instanceGroupName).Do()
			}
			if err != nil {

				logging.Error("Error shutting down instance group ", instanceGroupName, region, err)
				mutex.Lock()
				ret = append(ret, err)
				mutex.Unlock()
			}
			if err := WaitRegionInstanceGroupDeleted(singleZone, instanceGroupName, region, project, service); err != nil {
				logging.Error(err)
			}
			wg.Done()
		}(region)
	}
	wg.Wait()

	for _, igm := range igmList {
		for _, nxt := range igm {
			logging.Printf("Deleting disks from instance group %v, project %v", instanceGroupName, project)
			wg.Add(1)
			go func(nxt *compute.ManagedInstance) {
				zone, instName := getZoneAndName(nxt.Instance)

				if err := DoShutdownInstance(instName, project, zone, service); err != nil {
					logging.Error(err)
				}

				if err := DoDeleteDisk(instName, project, zone, service); err != nil {
					logging.Error(err)
					mutex.Lock()
					ret = append(ret, err)
					mutex.Unlock()
				}
				wg.Done()
			}(nxt)
		}
	}
	wg.Wait()

	return ret
}

// GetInstanceGroupIPs returns the list of ips for the regions, where each line rotates through the regions.
func GetInstanceGroupsIPs(singleZone bool, instanceGroupName, project string, regions []string,
	service *compute.Service) (ips []string, err error) {

	ipMap := make(map[string][]string)
	for _, region := range regions {

		var managedInstances []*compute.ManagedInstance
		if singleZone {
			igm, err := service.InstanceGroupManagers.ListManagedInstances(project, region, instanceGroupName).Do()
			if err != nil {
				logging.Error(err)
				return nil, err
			}
			managedInstances = igm.ManagedInstances
		} else {
			igm, err := service.RegionInstanceGroupManagers.ListManagedInstances(project, region, instanceGroupName).Do()
			if err != nil {
				logging.Error(err)
				return nil, err
			}
			managedInstances = igm.ManagedInstances
		}
		for _, nxt := range managedInstances {
			zone, instName := getZoneAndName(nxt.Instance)

			ist, err := service.Instances.Get(project, zone, instName).Do()
			if err != nil {
				logging.Error(err)
				return nil, err
			}
			if len(regions) == 1 {
				ipMap[region] = append(ipMap[region], ist.NetworkInterfaces[0].NetworkIP)
			} else {
				ipMap[region] = append(ipMap[region], ist.NetworkInterfaces[0].AccessConfigs[0].NatIP)
			}
		}
	}

	var maxLen int
	for _, nxtL := range ipMap {
		if len(nxtL) > maxLen {
			maxLen = len(nxtL)
		}
	}

	for i := 0; i < maxLen; i++ {
		for _, nxtL := range ipMap {
			if i < len(nxtL) {
				ips = append(ips, fmt.Sprintf("%v:%v", nxtL[i], Port))
			}
		}
	}

	return ips, nil
}

// DoRebootRegions reboots the nodes from the instance groups.
func DoRebootRegions(singleZone bool, instanceGroupName, project string, regions []string,
	service *compute.Service) (errs []error) {

	for _, region := range regions {

		var managedInstances []*compute.ManagedInstance
		if singleZone {
			igm, err := service.InstanceGroupManagers.ListManagedInstances(project, region, instanceGroupName).Do()
			if err != nil {
				logging.Error(err)
				return []error{err}
			}
			managedInstances = igm.ManagedInstances
		} else {
			igm, err := service.RegionInstanceGroupManagers.ListManagedInstances(project, region, instanceGroupName).Do()
			if err != nil {
				logging.Error(err)
				return []error{err}
			}
			managedInstances = igm.ManagedInstances
		}
		for _, nxt := range managedInstances {
			zone, instName := getZoneAndName(nxt.Instance)

			_, err := service.Instances.Reset(project, zone, instName).Do()
			if err != nil {
				logging.Error(err)
				errs = append(errs, err)
			}
		}
	}

	return errs
}

func LaunchInstanceGroups(singleZone bool, numNodes int, baseInstanceName, instanceGroupName,
	templateName, project string, regions []string,
	service *compute.Service) (err error) {

	logging.Print("Adding firewall rule")
	if err = DoAllFirewallRule(project, service); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			// shutdown the groups
			logging.Print("shutting down instance groups because of error")
			ShutdownInstanceGroups(singleZone, instanceGroupName, project, regions, service)
			logging.Print("removing firewall rule because of error")
			DoRemoveFirewallRule(project, service)
		}
	}()

	// if err = UpdateInstanceTemplateInstanceType(instanceType, templateName, project, service); err != nil {
	//	logging.Error(err)
	//	return
	// }
	igm := &compute.InstanceGroupManager{
		BaseInstanceName: baseInstanceName,
		InstanceTemplate: getMyTemplateURL(project, templateName),
		Name:             instanceGroupName,
		TargetSize:       int64(numNodes),
	}

	for _, region := range regions {
		logging.Printf("Launching instance group %v, using template %v, nodes %v, in zone %v, project %v",
			instanceGroupName, templateName, numNodes, region, project)
		if !singleZone {
			if _, err = service.RegionInstanceGroupManagers.Insert(project, region, igm).Do(); err != nil {
				logging.Error(err)
				return
			}
		} else {
			if service.InstanceGroupManagers.Insert(project, region, igm).Do(); err != nil {
				logging.Error(err)
				return
			}
		}
	}

	// wait for all to be ready
	for _, region := range regions {
		if err = WaitRegionInstanceGroupStable(singleZone, instanceGroupName, region, project, service); err != nil {
			return err
		}
	}
	return nil
}

func UpdateInstanceTemplateInstanceType(newInstanceType, templateName, project string,
	service *compute.Service) error {

	logging.Printf("Updating instance template %v with instance type %v", templateName, newInstanceType)
	// get the template
	it, err := service.InstanceTemplates.Get(project, templateName).Do()
	if err != nil {
		logging.Error(err)
		return err
	}

	// delete the existing template
	if err = DeleteInstanceTemplate(templateName, project, service); err != nil {
		logging.Error(err)
		return err
	}

	if it.Properties.MachineType == newInstanceType { // Already the correct machine type
		return nil
	}

	// update the instance type
	it.Properties.MachineType = newInstanceType

	if _, err = service.InstanceTemplates.Insert(project, it).Do(); err != nil {
		logging.Error(err)
		return err
	}

	// Wait for it to be ready
	if err = waitInstanceTemplateReady(templateName, project, service); err != nil {
		logging.Error(err)
		return err
	}
	return nil
}

func DeleteInstanceTemplate(templateName, project string, service *compute.Service) (err error) {
	logging.Printf("deleting instance template %v for project %v", templateName, project)
	if _, err := service.InstanceTemplates.Delete(project, templateName).Do(); err != nil {
		logging.Error(err)
		return err
	}
	// check it doesn't exits
	for true {
		if _, err := service.InstanceTemplates.Get(project, templateName).Do(); err == nil {
			logging.Print("waiting for instance template to be deleted")
			time.Sleep(retryTime)
		}
		break
	}
	return
}

func WaitRegionInstanceGroupDeleted(singleZone bool, instanceGroupName, region, project string, service *compute.Service) error {
	for {
		var igm *compute.InstanceGroupManager
		var err error
		if singleZone {
			igm, err = service.InstanceGroupManagers.Get(project, region, instanceGroupName).Do()
		} else {
			igm, err = service.RegionInstanceGroupManagers.Get(project, region, instanceGroupName).Do()
		}
		if err == nil {
			logging.Printf("waiting for instance group to shut down status: %v", igm.Status.ForceSendFields)
			time.Sleep(retryTime)
			continue
		}
		break
	}
	return nil
}

func WaitRegionInstanceGroupStable(singleZone bool, instanceGroupName, region, project string, service *compute.Service) error {
	for {
		var nxt *compute.InstanceGroupManager
		var err error
		if singleZone {
			nxt, err = service.InstanceGroupManagers.Get(project, region, instanceGroupName).Do()
		} else {
			nxt, err = service.RegionInstanceGroupManagers.Get(project, region, instanceGroupName).Do()
		}
		if err != nil {
			logging.Error(err)
			return err
		}
		if nxt.Status.IsStable {
			break
		}

		logging.Printf("Instance group manager %v:%v not stable, checking again", instanceGroupName, region)
		time.Sleep(retryTime)
	}

retryStatus:
	if singleZone {
		miList, err := service.InstanceGroupManagers.ListManagedInstances(project, region, instanceGroupName).Do()
		if err != nil {
			logging.Error(err)
			return err
		}
		for _, nxtI := range miList.ManagedInstances {

			if nxtI.InstanceStatus != "RUNNING" {
				logging.Print("Instance not yet running, checking again: ", nxtI.InstanceStatus, instanceGroupName, region)
				time.Sleep(retryTime)
				goto retryStatus
			}
		}
	} else {
		miList, err := service.RegionInstanceGroupManagers.ListManagedInstances(project, region, instanceGroupName).Do()
		if err != nil {
			logging.Error(err)
			return err
		}
		for _, nxtI := range miList.ManagedInstances {

			if nxtI.InstanceStatus != "RUNNING" {
				logging.Print("Instance not yet running, checking again: ", nxtI.InstanceStatus, instanceGroupName, region)
				time.Sleep(retryTime)
				goto retryStatus
			}
		}
	}
	return nil
}
