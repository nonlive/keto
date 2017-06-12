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
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/constants"
	cmdutil "github.com/UKHomeOffice/keto/pkg/keto/cmd/util"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:          "create <" + strings.Join(constants.ValidResourceTypes, "|") + "> <NAME>",
	Short:        "Create a resource",
	Long:         "Create a cluster or add a new computepool to existing cluster",
	ValidArgs:    constants.ValidResourceTypes,
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateCreateFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return runCreate(c, args)
	},
}

func validateCreateFlags(c *cobra.Command, args []string) error {
	validTypes := "Valid types: " + strings.Join(constants.ValidResourceTypes, ", ")

	if len(args) < 2 {
		return fmt.Errorf("resource type and or resource name not specified.\n" + validTypes)
	}

	if !cmdutil.StringInSlice(args[0], constants.ValidResourceTypes) {
		return fmt.Errorf("invalid resource type.\n" + validTypes)
	}

	// Check if mandatory flags are set when creating a computepool or a masterpool.
	if args[0] == constants.DefaultComputeName || args[0] == constants.DefaultMasterName {
		if !c.Flags().Changed("cluster") {
			return fmt.Errorf("cluster name must be set")
		}
	}

	// At this point cluster infra already exists, masterpool should be created
	// in the same networks, don't insist on needing the networks to be
	// specified when creating a masterpool.
	if args[0] != constants.DefaultMasterName {
		if !c.Flags().Changed("networks") {
			return fmt.Errorf("networks must be set")
		}
	}

	// TODO(vaijab): should not be required. Cloud providers could have sensible defaults.
	if !c.Flags().Changed("machine-type") {
		return fmt.Errorf("machine type must be set")
	}

	if !c.Flags().Changed("ssh-key") {
		return fmt.Errorf("ssh key must be set")
	}
	return nil
}

func runCreate(c *cobra.Command, args []string) error {
	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	resType := args[0]
	resName := args[1]

	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}
	coreOSVersion, err := c.Flags().GetString("coreos-version")
	if err != nil {
		return err
	}
	kubeVersion, err := c.Flags().GetString("kube-version")
	if err != nil {
		return err
	}
	sshKey, err := c.Flags().GetString("ssh-key")
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
	apiNetworks, err := c.Flags().GetStringSlice("api-networks")
	if err != nil {
		return err
	}
	networks, err := c.Flags().GetStringSlice("networks")
	if err != nil {
		return err
	}
	assetsDir, err := c.Flags().GetString("assets-dir")
	if err != nil {
		return err
	}

	if assetsDir == "" {
		d, err := os.Getwd()
		if err != nil {
			return nil
		}
		assetsDir = d
	}

	if resType == constants.DefaultClusterName {
		a, err := readAssetFiles(assetsDir)
		if err != nil {
			return err
		}
		cluster := model.Cluster{}
		cluster.Name = resName
		cluster.MasterPool = makeMasterPool("master", cluster.Name, coreOSVersion, kubeVersion, sshKey, machineType, diskSize, networks, apiNetworks)
		cluster.ComputePools = []model.ComputePool{makeComputePool("compute", cluster.Name, coreOSVersion, kubeVersion, sshKey, machineType, diskSize, networks)}

		return cli.ctrl.CreateCluster(cluster, a)
	}

	if resType == constants.DefaultMasterName {
		pool := makeMasterPool(resName, clusterName, coreOSVersion, kubeVersion, sshKey, machineType, diskSize, networks, apiNetworks)

		return cli.ctrl.CreateMasterPool(pool)
	}

	if resType == constants.DefaultComputeName {
		pool := makeComputePool(resName, clusterName, coreOSVersion, kubeVersion, sshKey, machineType, diskSize, networks)

		return cli.ctrl.CreateComputePool(pool)
	}

	return nil
}

// readAssetFiles reads asset files as byte arrays from the directory d and returns
// model.Assets.
func readAssetFiles(d string) (model.Assets, error) {
	var a model.Assets
	etcdCACertPath := path.Join(d, "etcd_ca.crt")
	etcdCAKeyPath := path.Join(d, "etcd_ca.key")
	kubeCACertPath := path.Join(d, "kube_ca.crt")
	kubeCAKeyPath := path.Join(d, "kube_ca.key")

	// Read in the certs.
	files := map[string]*[]byte{
		etcdCACertPath: &a.EtcdCACert,
		etcdCAKeyPath:  &a.EtcdCAKey,
		kubeCACertPath: &a.KubeCACert,
		kubeCAKeyPath:  &a.KubeCAKey,
	}
	for filename, v := range files {
		if !fileExists(filename) {
			return a, fmt.Errorf("%q does not exist", filename)
		}
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return a, err
		}
		*v = content
	}

	return a, nil
}

func fileExists(f string) bool {
	if _, err := os.Stat(f); os.IsNotExist(err) && err != nil {
		return false
	}
	return true
}

func makeMasterPool(name, clusterName, coreOSVersion, kubeVersion, sshKey, machineType string,
	diskSize int, networks, apiNetworks []string) model.MasterPool {

	p := model.MasterPool{}
	p.Name = name
	p.ClusterName = clusterName
	p.CoreOSVersion = coreOSVersion
	p.DiskSize = diskSize
	p.KubeAPINetworks = apiNetworks
	p.KubeVersion = kubeVersion
	p.MachineType = machineType
	p.Networks = networks
	p.SSHKey = sshKey
	// we default to using the master subnets unless specified
	if len(p.KubeAPINetworks) <= 0 {
		p.KubeAPINetworks = p.Networks
	}

	return p
}

func makeComputePool(name, clusterName, coreOSVersion, kubeVersion, sshKey, machineType string,
	diskSize int, networks []string) model.ComputePool {

	p := model.ComputePool{}
	p.ClusterName = clusterName
	p.CoreOSVersion = coreOSVersion
	p.DiskSize = diskSize
	p.KubeVersion = kubeVersion
	p.MachineType = machineType
	p.Name = name
	p.Networks = networks
	p.SSHKey = sshKey

	return p
}

func init() {
	RootCmd.AddCommand(createCmd)

	// Add flags that are relevant to create cmd.
	addAPINetworksFlag(createCmd)
	addAssetsDirFlag(createCmd)
	addClusterFlag(createCmd)
	addCoreOSVersionFlag(createCmd)
	addDiskSizeFlag(createCmd)
	addDNSZoneFlag(createCmd)
	addKubeVersionFlag(createCmd)
	addLabelsFlag(createCmd)
	addMachineTypeFlag(createCmd)
	addNetworksFlag(createCmd)
	addPoolSizeFlag(createCmd)
	addSSHKeyFlag(createCmd)
}
