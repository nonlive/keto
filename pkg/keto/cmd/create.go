/*
Copyright 2017 The Keto Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"strings"

	cmdutil "github.com/UKHomeOffice/keto/pkg/keto/cmd/util"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:          "create <" + strings.Join(resourceTypes, "|") + "> <NAME>",
	Short:        "Create a resource",
	Long:         "Create a cluster or add a new computepool to existing cluster",
	ValidArgs:    resourceTypes,
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateCreateFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return runCreate(c, args)
	},
}

func validateCreateFlags(c *cobra.Command, args []string) error {
	validTypes := "Valid types: " + strings.Join(resourceTypes, ", ")

	if len(args) < 2 {
		return fmt.Errorf("resource type and or resource name not specified.\n" + validTypes)
	}

	if !cmdutil.StringInSlice(args[0], resourceTypes) {
		return fmt.Errorf("invalid resource type.\n" + validTypes)
	}

	// Check if mandatory flags are set when creating a computepool
	if args[0] == "computepool" || args[0] == "masterpool" {
		if !c.Flags().Changed("cluster") {
			return fmt.Errorf("cluster name must be set")
		}
	}

	// At this point a cluster already exists, masterpool should be created in
	// the same networks.
	if args[0] != "masterpool" {
		if !c.Flags().Changed("networks") {
			return fmt.Errorf("networks must be set")
		}
	}

	// TODO(vaijab): should not be required. Cloud provivers could have sensible defaults.
	if !c.Flags().Changed("machine-type") {
		return fmt.Errorf("machine type must be set")
	}
	return nil
}

func runCreate(c *cobra.Command, args []string) error {
	client, err := newClient(c)
	if err != nil {
		return err
	}

	resType := args[0]
	resName := args[1]

	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}
	kubeVersion, err := c.Flags().GetString("kube-version")
	if err != nil {
		return err
	}
	machineType, err := c.Flags().GetString("machine-type")
	if err != nil {
		return err
	}
	diskSize, err := c.Flags().GetInt("disk-size")
	if err != nil {
		return err
	}
	networks, err := c.Flags().GetStringSlice("networks")
	if err != nil {
		return err
	}

	if resType == "cluster" {
		cluster := model.Cluster{}
		cluster.Name = resName
		cluster.MasterPool = makeMasterPool("master", resName, kubeVersion, machineType, networks, diskSize)

		if err := client.ctrl.CreateCluster(cluster); err != nil {
			return err
		}
	}

	if resType == "masterpool" {
		pool := model.MasterPool{}
		pool = makeMasterPool(resName, clusterName, kubeVersion, machineType, networks, diskSize)

		if err := client.ctrl.CreateMasterPool(pool); err != nil {
			return err
		}
	}

	if resType == "computepool" {
		// pool := model.ComputePool{}
		// pool.Name = resName
		// pool.ClusterName = clusterName

		// if err := client.ctrl.CreateComputePool(pool); err != nil {
		// 	return err
		// }
		return fmt.Errorf("not implemented")
	}

	return nil
}

func makeMasterPool(name, clusterName, kubeVersion, machineType string, networks []string, diskSize int) model.MasterPool {
	p := model.MasterPool{}
	p.Name = name
	p.ClusterName = clusterName
	p.KubeVersion = kubeVersion
	p.Networks = networks
	p.DiskSize = diskSize
	p.MachineType = machineType
	return p
}

func init() {
	RootCmd.AddCommand(createCmd)

	// Add flags that are relevant to create cmd.
	addClusterFlag(createCmd)
	addNetworksFlag(createCmd)
	addOSFlag(createCmd)
	addDiskSizeFlag(createCmd)
	addMachineTypeFlag(createCmd)
	addSizeFlag(createCmd)
	addDNSZoneFlag(createCmd)
	addLabelsFlag(createCmd)
	addKubeVersionFlag(createCmd)
}
