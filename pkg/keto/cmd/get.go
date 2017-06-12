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

	"github.com/UKHomeOffice/keto/pkg/constants"
	"github.com/UKHomeOffice/keto/pkg/keto"
	cmdutil "github.com/UKHomeOffice/keto/pkg/keto/cmd/util"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:          "get <" + strings.Join(constants.ValidResourceTypes, "|") + "> [NAME]",
	Short:        "Get a resource",
	Long:         "Get a resource",
	SuggestFor:   []string{"list"},
	ValidArgs:    constants.ValidResourceTypes,
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateGetFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return runGet(c, args)
	},
}

func validateGetFlags(c *cobra.Command, args []string) error {
	validTypes := "Valid types: " + strings.Join(constants.ValidResourceTypes, ", ")

	if len(args) < 1 {
		return fmt.Errorf("resource type not specified. " + validTypes)
	}

	if !cmdutil.StringInSlice(args[0], constants.ValidResourceTypes) {
		return fmt.Errorf("invalid resource type. " + validTypes)
	}
	return nil
}

func runGet(c *cobra.Command, args []string) error {
	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}
	res := args[0]
	resName := ""
	if len(args) == 2 {
		resName = args[1]
	}

	switch res {
	case constants.DefaultClusterName:
		return listClusters(cli, resName)
	case constants.DefaultMasterName:
		return listMasterPools(cli, clusterName, resName)
	case constants.DefaultComputeName:
		return listComputePools(cli, clusterName, resName)
	}
	return nil
}

func listMasterPools(cli *cli, clusterName, name string) error {
	pools, err := cli.ctrl.GetMasterPools(clusterName, name)
	if err != nil {
		return err
	}
	return keto.PrintMasterPool(keto.GetPrinter(os.Stdout), pools, true)
}

func listComputePools(cli *cli, clusterName, name string) error {
	pools, err := cli.ctrl.GetComputePools(clusterName, name)
	if err != nil {
		return err
	}
	return keto.PrintComputePool(keto.GetPrinter(os.Stdout), pools, true)
}

func listClusters(cli *cli, name string) error {
	clusters, err := cli.ctrl.GetClusters(name)
	if err != nil {
		return err
	}
	return keto.PrintClusters(keto.GetPrinter(os.Stdout), clusters, true)
}

func init() {
	RootCmd.AddCommand(getCmd)
	addClusterFlag(getCmd)
}
