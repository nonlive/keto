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
	"strings"

	cmdutil "github.com/UKHomeOffice/keto/pkg/keto/cmd/util"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:          "create <" + strings.Join(resourceTypes, "|") + "> <name>",
	Short:        "Create a resource",
	Long:         "Create a resource",
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

	// Check if mandatory flags are set when creating a nodepool
	if args[0] == "nodepool" {
		if !c.Flags().Changed("cluster") {
			return fmt.Errorf("cluster name must be set")

		}
		// TODO: add other flags validation
	}
	return nil
}

func runCreate(c *cobra.Command, args []string) error {
	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	res := args[0]
	resName := args[1]

	if res == "nodepool" {
		if err := client.ctrl.CreateNodePool(clusterName, resName); err != nil {
			return err
		}
	} else {
		// TODO: implement creating clusters
		return errors.New("not implemented")
	}

	return nil
}

func init() {
	RootCmd.AddCommand(createCmd)

	// Add flags that are relevant to create cmd.
	addClusterFlag(createCmd)
	addNetworksFlag(createCmd)
	addKindFlag(createCmd)
	addOSFlag(createCmd)
	addMachineTypeFlag(createCmd)
	addSizeFlag(createCmd)
	addDNSZoneFlag(createCmd)
	addLabelsFlag(createCmd)
	addKubeVersionFlag(createCmd)
}
