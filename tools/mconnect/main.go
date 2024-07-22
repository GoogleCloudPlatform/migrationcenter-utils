/*
Copyright 2024 Google LLC All Rights Reserved.

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

// mconnect is used to export and merge information from Migration Center and CAST to BigQuery, which allows to perform data analysis in Looker Studio.
package main

import (
	_ "github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/commands/export"
	_ "github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/commands/groups"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/commands/root"
	_ "github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/commands/views"
	"github.com/spf13/cobra"
)

func main() {
	walk(root.RootCmd, func(c *cobra.Command) {
		// Setting the help flag for the root command and its children.
		c.Flags().BoolP("help", "h", false, "Help for "+c.Name())
	})

	root.Execute()
}

// walk calls f for c and all of its children.
func walk(c *cobra.Command, f func(*cobra.Command)) {
	f(c)
	for _, c := range c.Commands() {
		walk(c, f)
	}
}
