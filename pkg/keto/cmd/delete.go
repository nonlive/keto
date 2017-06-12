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

	"github.com/UKHomeOffice/keto/pkg/constants"
	cmdutil "github.com/UKHomeOffice/keto/pkg/keto/cmd/util"

	"github.com/spf13/cobra"
)

// deleteCmd represents the create command
var deleteCmd = &cobra.Command{
	Use:          "delete <" + strings.Join(constants.ValidResourceTypes, "|") + "> <NAME>",
	Short:        "Delete a resource",
	Long:         "Delete a resource",
	SuggestFor:   []string{"remove"},
	ValidArgs:    constants.ValidResourceTypes,
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateDeleteFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return runDelete(c, args)
	},
}

func validateDeleteFlags(c *cobra.Command, args []string) error {
	validTypes := "Valid types: " + strings.Join(constants.ValidResourceTypes, ", ")

	if len(args) < 2 {
		return fmt.Errorf("resource type and or resource name not specified.\n" + validTypes)
	}

	if !cmdutil.StringInSlice(args[0], constants.ValidResourceTypes) {
		return fmt.Errorf("invalid resource type.\n" + validTypes)
	}

	// Check if mandatory flags are set when deleting a computepool or a masterpool.
	if args[0] == constants.DefaultComputeName || args[0] == constants.DefaultMasterName {
		if !c.Flags().Changed("cluster") {
			return fmt.Errorf("cluster name must be set")
		}
	}

	return nil
}

func runDelete(c *cobra.Command, args []string) error {
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

	switch resType {
	case "cluster":
		return cli.ctrl.DeleteCluster(resName)
	case constants.DefaultMasterName:
		return cli.ctrl.DeleteMasterPool(clusterName)
	case constants.DefaultComputeName:
		return cli.ctrl.DeleteComputePool(clusterName, resName)
	}

	return nil
}

func init() {
	RootCmd.AddCommand(deleteCmd)

	// Add flags that are relevant to delete cmd.
	addClusterFlag(deleteCmd)
}
