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

// Package groups contains the implmentation of the create-groups command used to create a group for each CAST application
// in Migration Center and add the 'mconnect' label to it.
package groups

import (
	"context"
	"fmt"
	"strings"

	migrationcenter "cloud.google.com/go/migrationcenter/apiv1"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/mcutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/messages"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/parser"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/commands/root"
	"github.com/googleapis/gax-go/v2"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
	st "google.golang.org/grpc/status"

	migrationcenterpb "cloud.google.com/go/migrationcenter/apiv1/migrationcenterpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	// value according to google.golang.org/grpc/codes
	alreadyExistCode = 6
	LabelKey         = "mconnect"
	labelValue       = "mconnect"
)

var (
	path                 string
	projectID            string
	region               string
	endpoint             string
	ignoreExistingGroups bool
)

type groupsClient interface {
	CreateGroup(ctx context.Context, req *migrationcenterpb.CreateGroupRequest, opts ...gax.CallOption) (*migrationcenter.CreateGroupOperation, error)
	GetGroup(ctx context.Context, req *migrationcenterpb.GetGroupRequest, opts ...gax.CallOption) (*migrationcenterpb.Group, error)
	UpdateGroup(ctx context.Context, req *migrationcenterpb.UpdateGroupRequest, opts ...gax.CallOption) (*migrationcenter.UpdateGroupOperation, error)
	Close() error
}

type groupCreatorFactory interface {
	build(pal mcutil.ProjectAndLocation, client groupsClient) groupCreator
}

type groupCreator interface {
	create(ctx context.Context, groups []string, ignoreExist bool) error
}

type mcGroupCreator struct {
	pal    mcutil.ProjectAndLocation
	client groupsClient
}

func (gc *mcGroupCreator) build(pal mcutil.ProjectAndLocation, client groupsClient) groupCreator {
	gc.pal = pal
	gc.client = client
	return gc
}

// create creates Migration Center groups for each group sequentially. It also adds a 'mconnect' label to each group created.
func (gc *mcGroupCreator) create(ctx context.Context, groups []string, ignoreExist bool) error {

	label := map[string]string{LabelKey: labelValue}
	for _, group := range groups {
		// Prepare group details -
		groupDetails := &migrationcenterpb.Group{
			Name:        group,
			DisplayName: group,
			Description: fmt.Sprintf("%v application group. Created using mconnect.", group),
			Labels:      label,
		}

		// Migration Center groups need to be lower and without spaces.
		groupID, diff := format(group)
		if diff {
			fmt.Printf("Spaces or underscores were found for application '%v', groupID '%v' will be used.\n", group, groupID)
		}
		req := &migrationcenterpb.CreateGroupRequest{Parent: gc.pal.Path(), GroupId: groupID, Group: groupDetails}
		op, err := gc.client.CreateGroup(ctx, req)
		if err != nil {
			if alreadyExist(err) && ignoreExist {
				if err := gc.updateLabel(ctx, groupID); err != nil {
					return err
				}
				continue
			}
			return fmt.Errorf("failed creating group: '%v', err: %v", group, err)
		}
		_, err = op.Wait(ctx)
		if err != nil {
			return fmt.Errorf("failed creating group: '%v', err: %v", group, err)
		}
		fmt.Println(messages.GroupCreated{Group: groupID})
	}

	return nil
}

func (gc *mcGroupCreator) updateLabel(ctx context.Context, groupID string) error {
	groupPath := gc.pal.Path() + "/groups/" + groupID
	getReq := &migrationcenterpb.GetGroupRequest{Name: groupPath}
	oldGroup, err := gc.client.GetGroup(ctx, getReq)
	if err != nil {
		return err
	}

	labels := oldGroup.Labels
	// If there are no labels the Labels field will be nil
	if labels == nil {
		labels = make(map[string]string)
	}
	// Check if this group was already created by this tool.
	if _, ok := labels[LabelKey]; ok {
		fmt.Println(messages.GroupDetectedLabel{Group: groupID, Label: LabelKey})
		return nil
	}
	labels[LabelKey] = labelValue

	updateReq := &migrationcenterpb.UpdateGroupRequest{UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}}, Group: &migrationcenterpb.Group{Name: groupPath, Labels: labels}}
	op, err := gc.client.UpdateGroup(ctx, updateReq)
	if err != nil {
		return fmt.Errorf("failed updating group: '%v', err: %v", groupID, err)
	}
	_, err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed updating group: '%v', err: %v", groupID, err)
	}
	fmt.Println(messages.GroupUpdated{Name: groupID})
	return nil
}

// NewCreateGroupsCmd returns the createGroups command.
func NewCreateGroupsCmd(factory groupCreatorFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "create-groups path project region",
		Short: "Creates a group for each CAST application in Migration Center and adds the 'mconnect' label to it.",
		Long:  `Creates a group for each CAST application in Migration Center and adds the 'mconnect' label to it.`,
		Example: `	
		mconnect create-groups --path=path/to/cast/analysisResults.csv --project=my-mc-project-id --region=my-region1
		mconnect create-groups --path=path/to/cast/analysisResults.csv --project=my-mc-project-id --region=my-region1 --ignore-existing-groups=true`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if len(args) != 0 {
				return messages.NoArgumentsAcceptedError{Args: args}.Error()
			}

			fmt.Println(messages.ParsingFile{FilePath: path})
			groups, err := parser.Groups(path)
			if err != nil {
				return fmt.Errorf("failed parsing, err: %v", err)
			}
			fmt.Println(messages.GroupsApps{Applications: len(groups)})

			location := root.DefaultLocation
			if region != "" {
				location = region
			}

			fmt.Println(messages.GroupsCreation)
			opts := []option.ClientOption{option.WithUserAgent(messages.GroupsUserAgent)}
			if endpoint != "" {
				opts = append(opts, option.WithEndpoint(endpoint))
			}

			client, err := migrationcenter.NewClient(ctx, opts...)
			if err != nil {
				return err
			}
			defer client.Close()

			pal := mcutil.ProjectAndLocation{ProjectID: projectID, Location: location}
			gc := factory.build(pal, client)
			err = gc.create(ctx, groups, ignoreExistingGroups)
			if err != nil {
				return fmt.Errorf("failed creating groups, err: %v", err)
			}

			fmt.Println(messages.GroupsSuccess)
			fmt.Println(messages.GroupsNextSteps{Path: path, ProjectID: projectID, Region: location})
			return nil
		},
	}
}

func init() {
	createGroupsCmd := NewCreateGroupsCmd(&mcGroupCreator{})
	// Required flags.
	createGroupsCmd.Flags().StringVar(&projectID, "project", "", `The project-id in which to create the Migration Center groups. Make sure to use the same Project ID for every command. (required)`)
	createGroupsCmd.MarkFlagRequired("project")
	createGroupsCmd.Flags().StringVar(&path, "path", "", `The csv file's path which contains CAST's report (analysisResults.csv). (required)`)
	createGroupsCmd.MarkFlagRequired("path")
	createGroupsCmd.Flags().StringVar(&region, "region", "", `The Migration Center region in which the groups will be created. (required)`)
	createGroupsCmd.MarkFlagRequired("region")

	// Optional flags.
	createGroupsCmd.Flags().BoolVarP(&ignoreExistingGroups, "ignore-existing-groups", "i", false, `Continue if mconnect is trying to create a group that already exists in Migration Center.
If set to 'true', the 'mconnect' label will be added to every group that already exists as well.`)
	createGroupsCmd.Flags().StringVar(&endpoint, "mc-endpoint", "", `The endpoint Migration Centers client will use.`)
	createGroupsCmd.Flags().MarkHidden("mc-endpoint")
	root.RootCmd.AddCommand(createGroupsCmd)
}

func alreadyExist(err error) bool {
	if err == nil {
		return false
	}
	return st.Code(err) == alreadyExistCode
}

func format(group string) (string, bool) {
	lowGroup := strings.ToLower(group)
	groupID := strings.ReplaceAll(lowGroup, " ", "")
	groupID = strings.Replace(groupID, "_", "-", -1)
	return groupID, lowGroup != groupID
}
