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

	"github.com/spf13/cobra"
)

// deleteCmd represents the create command
var deleteCmd = &cobra.Command{
	Use:          "delete <subcommand>",
	Short:        "Delete resources",
	SuggestFor:   []string{"remove"},
	SilenceUsage: true,
}

var deleteClusterCmd = &cobra.Command{
	Use:          "cluster <NAME>",
	Aliases:      clusterCmdAliases,
	Short:        "Delete a cluster",
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		return deleteClusterCmdFunc(c, args)
	},
}

func deleteClusterCmdFunc(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("cluster name is not specified")
	}

	cli, err := newCLI(c)
	if err != nil {
		return err
	}

	cli.logger.Printf("Deleting cluster %q", args)
	if err := cli.ctrl.DeleteCluster(args...); err != nil {
		return err
	}
	cli.logger.Printf("Cluster %q successfully deleted", args)
	return nil
}

var deleteMasterPoolCmd = &cobra.Command{
	Use:          "masterpool",
	Aliases:      masterPoolCmdAliases,
	Short:        "Delete a masterpool",
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateDeleteFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return deleteMasterPoolCmdFunc(c, args)
	},
}

func deleteMasterPoolCmdFunc(c *cobra.Command, args []string) error {
	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}

	cli, err := newCLI(c)
	if err != nil {
		return err
	}
	cli.logger.Printf("Deleting masterpool of cluster %q", clusterName)
	if err := cli.ctrl.DeleteMasterPool(clusterName); err != nil {
		return err
	}
	cli.logger.Printf("Masterpool successfully deleted")
	return nil
}

var deleteComputePoolCmd = &cobra.Command{
	Use:          "computepool <NAME>",
	Aliases:      computePoolCmdAliases,
	Short:        "Delete a computepool",
	SilenceUsage: true,
	PreRunE: func(c *cobra.Command, args []string) error {
		return validateDeleteFlags(c, args)
	},
	RunE: func(c *cobra.Command, args []string) error {
		return deleteComputePoolCmdFunc(c, args)
	},
}

func deleteComputePoolCmdFunc(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("computepool name is not specified")
	}

	clusterName, err := c.Flags().GetString("cluster")
	if err != nil {
		return err
	}

	cli, err := newCLI(c)
	if err != nil {
		return err
	}
	cli.logger.Printf("Deleting computepool %q of cluster %q", args, clusterName)
	if err := cli.ctrl.DeleteComputePool(clusterName, args...); err != nil {
		return err
	}
	cli.logger.Printf("Computepool %q successfully deleted", args)
	return nil
}

func validateDeleteFlags(c *cobra.Command, args []string) error {
	// Check if cluster name has been set. TODO(vaijab): should controller take
	// care of validation?
	if c.Name() == "masterpool" || c.Name() == "computepool" {
		if !c.Flags().Changed("cluster") {
			return fmt.Errorf("cluster name must be set")
		}
	}
	return nil
}

func init() {
	deleteCmd.AddCommand(
		deleteClusterCmd,
		deleteMasterPoolCmd,
		deleteComputePoolCmd,
	)

	// Add flags that are relevant to delete subcommands.
	addClusterFlag(
		deleteMasterPoolCmd,
		deleteComputePoolCmd,
	)
}
