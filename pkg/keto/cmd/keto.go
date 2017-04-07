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
	"os"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/controller"
	"github.com/UKHomeOffice/keto/pkg/userdata"

	"github.com/spf13/cobra"
)

var (
	// resourceTypes contains a list of currently supported resource types.
	resourceTypes = []string{"cluster", "masterpool", "computepool"}

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "keto",
		Short: "Kubernetes clusters manager",
		Long:  "Kubernetes clusters manager",
		RunE: func(c *cobra.Command, args []string) error {
			if c.Flags().Changed("version") {
				printVersion()
				return nil
			}
			return c.Usage()
		},
	}
)

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

// client respresents keto cli client structure.
type client struct {
	ctrl *controller.Controller
}

// newClient returns a new instance of client. It is expected to be used by
// keto cli subcommands.
func newClient(c *cobra.Command) (*client, error) {
	if !c.Flags().Changed("cloud") {
		return &client{}, fmt.Errorf("cloud provider name is not specified")
	}
	cloudName, err := c.Flags().GetString("cloud")
	if err != nil {
		return &client{}, err
	}

	cloud, err := cloudprovider.InitCloudProvider(cloudName, nil)
	if err != nil {
		return &client{}, err
	}
	ud := userdata.New()
	ctrl := controller.New(
		controller.Config{
			Cloud:    cloud,
			UserData: ud,
		})

	return &client{ctrl: ctrl}, nil
}

func init() {
	// Local flags
	RootCmd.Flags().BoolP("help", "h", false, "Help message")
	RootCmd.Flags().BoolP("version", "v", false, "Print version")

	// Global flags
	RootCmd.PersistentFlags().String("cloud", "",
		"Cloud provider name. Supported providers: "+strings.Join(cloudprovider.CloudProviders(), ", "))
	RootCmd.PersistentFlags().String("cloud-config", "", "Cloud provider config file")
}

// addClusterFlag adds a cluster flag
func addClusterFlag(c *cobra.Command) {
	c.Flags().String("cluster", "", "Cluster name")
}

// addNetworksFlag adds a networks flag
func addNetworksFlag(c *cobra.Command) {
	c.Flags().StringSlice("networks", []string{}, "Cloud specific list of comma separated networks")
}

// addOSFlag adds an OS flag
func addOSFlag(c *cobra.Command) {
	c.Flags().String("os", "", "Operating system")
}

// addSSHKeyFlag adds an ssh-key flag
func addSSHKeyFlag(c *cobra.Command) {
	c.Flags().String("ssh-key", "", "Public SSH key or name (dependent on cloud provider)")
}

// addDiskSizeFlag adds a disk-size flag
func addDiskSizeFlag(c *cobra.Command) {
	c.Flags().Int("disk-size", 10, "Node boot disk size in GB")
}

// addMachineTypeFlag adds a machine type flag
func addMachineTypeFlag(c *cobra.Command) {
	c.Flags().String("machine-type", "", "Machine type")
}

// addSizeFlag adds a size flag
func addSizeFlag(c *cobra.Command) {
	c.Flags().Int("size", 0, "Number of nodes in the compute pool")
}

// addDNSZoneFlag adds a DNS zone flag
func addDNSZoneFlag(c *cobra.Command) {
	c.Flags().String("dns-zone", "", "Hosted DNS zone name")
}

// addLabelsFlag adds labels flag
func addLabelsFlag(c *cobra.Command) {
	c.Flags().StringSlice("labels", []string{}, "List of labels in a comma separated key=value format")
}

// addKubeVersionFlag adds a kubernetes version flag
func addKubeVersionFlag(c *cobra.Command) {
	c.Flags().String("kube-version", "v1.6.0-rc.1", "Kubernetes version")
}
