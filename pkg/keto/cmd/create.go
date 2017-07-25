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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/UKHomeOffice/keto/pkg/keto/util"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/spf13/cobra"
)

// createCmd represents the 'create' command
var createCmd = &cobra.Command{
	Use:          "create <subcommand>",
	Short:        "Create a resource",
	Long:         "Create a new resource",
	SilenceUsage: true,
}

var createClusterCmd = &cobra.Command{
	Use:          "cluster NAME",
	Aliases:      clusterCmdAliases,
	Short:        "Create a cluster",
	Long:         "Create a cluster",
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateCreateFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return createClusterCmdFunc(c, args)
	},
}

func validateCreateFlags(c *cobra.Command, args []string) error {
	// Check if cluster name has been set. TODO(vaijab): should controller take
	// care of validation?
	if c.Name() == "masterpool" || c.Name() == "computepool" {
		if !c.Flags().Changed("cluster") {
			return fmt.Errorf("cluster name must be set")
		}
	}

	// TODO(vaijab): should not be required. Cloud providers could have
	// sensible defaults, the logic should live in the controller though.
	if !c.Flags().Changed("machine-type") {
		return fmt.Errorf("machine type must be set")
	}

	// TODO(vaijab): should default to master ssh-key when creating compute
	// pools if not specified, the logic should live in the controller though.
	if !c.Flags().Changed("ssh-key") {
		return fmt.Errorf("ssh key must be set")
	}
	return nil
}

func createClusterCmdFunc(c *cobra.Command, args []string) error {
	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	if len(args) != 1 {
		return errors.New("cluster name is not specified")
	}
	name := args[0]

	assetsDir, err := c.Flags().GetString("assets-dir")
	if err != nil {
		return err
	}
	if assetsDir == "" {
		d, err := os.Getwd()
		if err != nil {
			return err
		}
		assetsDir = d
		cli.debugLogger.Printf("assets directory is not specified, using %q instead", assetsDir)
	}
	a, err := cli.readAssetFiles(assetsDir)
	if err != nil {
		return err
	}

	cluster := model.Cluster{}
	cluster.Name = name

	// Let controller ensure node pools are marked internal or external
	// depending on cluster.Internal flag.
	internal, err := c.Flags().GetBool("internal")
	if err != nil {
		return err
	}
	cluster.Internal = internal

	// DNSZone is not required.
	dnsZone, err := c.Flags().GetString("dns-zone")
	if err != nil {
		return err
	}
	cluster.DNSZone = dnsZone

	labels, err := c.Flags().GetStringSlice("labels")
	if err != nil {
		return err
	}
	cluster.Labels = util.KVsToLabels(labels)

	p, err := makeMasterPool("master", name, *c)
	if err != nil {
		return err
	}
	cluster.MasterPool = p

	numComputePools, err := c.Flags().GetInt("compute-pools")
	if err != nil {
		return err
	}
	for i := 0; i < numComputePools; i++ {
		p, err := makeComputePool("compute"+strconv.Itoa(i), name, *c)
		if err != nil {
			return err
		}
		cluster.ComputePools = append(cluster.ComputePools, p)
	}

	cli.logger.Printf("Creating cluster %q", cluster.Name)
	if err := cli.ctrl.CreateCluster(cluster, a); err != nil {
		return err
	}
	cli.logger.Printf("Cluster %q successfully created", cluster.Name)
	return nil
}

// readAssetFiles reads asset files as byte arrays from the directory d and returns
// model.Assets.
func (c cli) readAssetFiles(d string) (model.Assets, error) {
	a := model.Assets{}
	etcdCACertPath := path.Join(d, "etcd_ca.crt")
	etcdCAKeyPath := path.Join(d, "etcd_ca.key")
	kubeCACertPath := path.Join(d, "kube_ca.crt")
	kubeCAKeyPath := path.Join(d, "kube_ca.key")

	// Check if assets exists.
	if !fileExists(d) {
		return a, fmt.Errorf("assets directory %q does not exist", d)
	}
	if !fileExists(etcdCACertPath) {
		return a, fmt.Errorf("%q does not exist", etcdCACertPath)
	}
	if !fileExists(etcdCAKeyPath) {
		return a, fmt.Errorf("%q does not exist", etcdCAKeyPath)
	}
	if !fileExists(kubeCACertPath) {
		return a, fmt.Errorf("%q does not exist", kubeCACertPath)
	}
	if !fileExists(kubeCAKeyPath) {
		return a, fmt.Errorf("%q does not exist", kubeCAKeyPath)
	}

	// Read etcd CA cert.
	c.debugLogger.Printf("reading assets file %q", etcdCACertPath)
	etcdCACert, err := ioutil.ReadFile(etcdCACertPath)
	if err != nil {
		return a, err
	}
	a.EtcdCACert = etcdCACert

	// Read etcd CA key.
	c.debugLogger.Printf("reading assets file %q", etcdCAKeyPath)
	etcdCAKey, err := ioutil.ReadFile(etcdCAKeyPath)
	if err != nil {
		return a, err
	}
	a.EtcdCAKey = etcdCAKey

	// Read kube CA cert.
	c.debugLogger.Printf("reading assets file %q", kubeCACertPath)
	kubeCACert, err := ioutil.ReadFile(kubeCACertPath)
	if err != nil {
		return a, err
	}
	a.KubeCACert = kubeCACert

	// Read kube CA key.
	c.debugLogger.Printf("reading assets file %q", kubeCAKeyPath)
	kubeCAKey, err := ioutil.ReadFile(kubeCAKeyPath)
	if err != nil {
		return a, err
	}
	a.KubeCAKey = kubeCAKey

	return a, nil
}

func fileExists(f string) bool {
	if _, err := os.Stat(f); os.IsNotExist(err) && err != nil {
		return false
	}
	return true
}

var createMasterPoolCmd = &cobra.Command{
	Use:          "masterpool NAME",
	Aliases:      masterPoolCmdAliases,
	Short:        "Create a masterpool",
	Long:         "Create a masterpool",
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateCreateFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return createMasterPoolCmdFunc(c, args)
	},
}

func createMasterPoolCmdFunc(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("masterpool name is not specified")
	}
	name := args[0]

	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}
	p, err := makeMasterPool(name, clusterName, *c)
	if err != nil {
		return err
	}

	cli, err := newCLI(c)
	if err != nil {
		return err
	}
	cli.logger.Printf("Creating masterpool %q for cluster %q", p.Name, p.ClusterName)
	if err := cli.ctrl.CreateMasterPool(p); err != nil {
		return err
	}
	cli.logger.Printf("Masterpool %q successfully created", p.Name)
	return nil
}

func makeMasterPool(name, clusterName string, c cobra.Command) (model.MasterPool, error) {
	p := model.MasterPool{}

	coreOSVersion, err := c.Flags().GetString("coreos-version")
	if err != nil {
		return p, err
	}
	kubeVersion, err := c.Flags().GetString("kube-version")
	if err != nil {
		return p, err
	}
	sshKey, err := c.Flags().GetString("ssh-key")
	if err != nil {
		return p, err
	}
	machineType, err := c.Flags().GetString("machine-type")
	if err != nil {
		return p, err
	}
	diskSize, err := c.Flags().GetInt("disk-size")
	if err != nil {
		return p, err
	}
	networks, err := c.Flags().GetStringSlice("networks")
	if err != nil {
		return p, err
	}
	labels, err := c.Flags().GetStringSlice("labels")
	if err != nil {
		return p, err
	}
	p.Labels = util.KVsToLabels(labels)

	p.Name = name
	p.ClusterName = clusterName
	p.CoreOSVersion = coreOSVersion
	p.KubeVersion = kubeVersion
	p.SSHKey = sshKey
	p.Networks = networks
	p.DiskSize = diskSize
	p.MachineType = machineType
	return p, nil
}

var createComputePoolCmd = &cobra.Command{
	Use:          "computepool NAME",
	Aliases:      computePoolCmdAliases,
	Short:        "Create a computepool",
	Long:         "Create a computepool",
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateCreateFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return createComputePoolCmdFunc(c, args)
	},
}

func createComputePoolCmdFunc(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("computepool name is not specified")
	}
	name := args[0]

	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}
	p, err := makeComputePool(name, clusterName, *c)
	if err != nil {
		return err
	}

	cli, err := newCLI(c)
	if err != nil {
		return err
	}
	cli.logger.Printf("Creating computepool %q for cluster %q", p.Name, p.ClusterName)
	if err := cli.ctrl.CreateComputePool(p); err != nil {
		return err
	}
	cli.logger.Printf("Masterpool %q successfully created", p.Name)
	return nil
}

func makeComputePool(name, clusterName string, c cobra.Command) (model.ComputePool, error) {
	p := model.ComputePool{}

	coreOSVersion, err := c.Flags().GetString("coreos-version")
	if err != nil {
		return p, err
	}
	kubeVersion, err := c.Flags().GetString("kube-version")
	if err != nil {
		return p, err
	}
	sshKey, err := c.Flags().GetString("ssh-key")
	if err != nil {
		return p, err
	}
	machineType, err := c.Flags().GetString("machine-type")
	if err != nil {
		return p, err
	}
	size, err := c.Flags().GetInt("pool-size")
	if err != nil {
		return p, err
	}
	diskSize, err := c.Flags().GetInt("disk-size")
	if err != nil {
		return p, err
	}

	networks, err := c.Flags().GetStringSlice("compute-networks")
	if err != nil {
		return p, err
	}

	if len(networks) == 0 {
		networks, err = c.Flags().GetStringSlice("networks")
		if err != nil {
			return p, err
		}
	}

	labels, err := c.Flags().GetStringSlice("labels")
	if err != nil {
		return p, err
	}
	p.Labels = util.KVsToLabels(labels)

	p.Name = name
	p.ClusterName = clusterName
	p.CoreOSVersion = coreOSVersion
	p.KubeVersion = kubeVersion
	p.SSHKey = sshKey
	p.Networks = networks
	p.DiskSize = diskSize
	p.MachineType = machineType
	p.Size = size
	return p, nil
}

func init() {
	createCmd.AddCommand(
		createClusterCmd,
		createMasterPoolCmd,
		createComputePoolCmd,
	)

	// Add flags that are relevant to different subcommands.
	addClusterFlag(
		createMasterPoolCmd,
		createComputePoolCmd,
	)

	addInternalFlag(
		createClusterCmd,
	)

	addNetworksFlag(
		createClusterCmd,
		createMasterPoolCmd,
		createComputePoolCmd,
	)

	addCoreOSVersionFlag(
		createClusterCmd,
		createMasterPoolCmd,
		createComputePoolCmd,
	)

	addSSHKeyFlag(
		createClusterCmd,
		createMasterPoolCmd,
		createComputePoolCmd,
	)

	addDiskSizeFlag(
		createClusterCmd,
		createMasterPoolCmd,
		createComputePoolCmd,
	)

	addMachineTypeFlag(
		createClusterCmd,
		createMasterPoolCmd,
		createComputePoolCmd,
	)

	addLabelsFlag(
		createClusterCmd,
		createComputePoolCmd,
		createMasterPoolCmd,
	)

	addKubeVersionFlag(
		createClusterCmd,
		createComputePoolCmd,
		createMasterPoolCmd,
	)

	addPoolSizeFlag(
		createClusterCmd,
		createComputePoolCmd,
	)

	addComputePoolsFlag(
		createClusterCmd,
	)

	addComputeNetworksFlag(
		createClusterCmd,
	)

	addAssetsDirFlag(
		createClusterCmd,
	)

	addDNSZoneFlag(
		createClusterCmd,
	)
}
