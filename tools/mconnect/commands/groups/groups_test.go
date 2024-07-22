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

package groups

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"

	migrationcenter "cloud.google.com/go/migrationcenter/apiv1"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/gapiutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/mcutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	migrationcenterpb "cloud.google.com/go/migrationcenter/apiv1/migrationcenterpb"
	"google.golang.org/protobuf/types/known/anypb"
)

type fakeMCService struct {
	migrationcenterpb.UnimplementedMigrationCenterServer
	groups map[string]*migrationcenterpb.Group
}

func (fm *fakeMCService) initGroups(groups []*migrationcenterpb.Group) {
	fm.groups = make(map[string]*migrationcenterpb.Group)
	for _, group := range groups {
		fm.groups[strings.ToLower(group.Name)] = group
	}
}

func (fm *fakeMCService) CreateGroup(_ context.Context, req *migrationcenterpb.CreateGroupRequest) (*longrunningpb.Operation, error) {
	if _, ok := fm.groups[req.Group.Name]; ok {
		return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("group '%v' already exists", req.Group.Name))
	}

	group := &migrationcenterpb.Group{Name: req.Group.Name, DisplayName: req.Group.DisplayName, Labels: req.Group.Labels, Description: req.Group.Description}
	fm.groups[strings.ToLower(req.Group.Name)] = group

	groupPb, err := anypb.New(group)
	if err != nil {
		return nil, fmt.Errorf("fakeMCService failed creating group '%v', err: %v", req.Group.Name, err)
	}
	return &longrunningpb.Operation{Name: fmt.Sprintf("operations/testGroup%v", req.Group.Name), Done: true, Result: &longrunningpb.Operation_Response{Response: groupPb}}, nil

}

func (fm *fakeMCService) GetGroup(_ context.Context, req *migrationcenterpb.GetGroupRequest) (*migrationcenterpb.Group, error) {
	// A request names is built as "path/pathID/location/location/groups/groupID" - hence we split on groups to get the group name.
	groupName := strings.Split(req.Name, "/groups/")[1]
	if group, ok := fm.groups[strings.ToLower(groupName)]; ok {
		return group, nil
	}
	return nil, fmt.Errorf("group '%v' doesn't exist", groupName)
}

func (fm *fakeMCService) UpdateGroup(_ context.Context, req *migrationcenterpb.UpdateGroupRequest) (*longrunningpb.Operation, error) {
	paths := req.GetUpdateMask().Paths
	if len(paths) != 1 || paths[0] != "labels" {
		return nil, fmt.Errorf("expected path to contain only: 'labels', contains: '%v'", paths)
	}

	groupName := strings.Split(req.Group.Name, "/groups/")[1]
	group, ok := fm.groups[groupName]
	if !ok {
		return nil, fmt.Errorf("couldn't update group '%v', group doesn't exist", groupName)
	}
	group.Labels = req.Group.Labels
	fm.groups[groupName] = group

	groupPb, err := anypb.New(group)
	if err != nil {
		return nil, fmt.Errorf("failed to update group '%v', err: %v", groupName, err)
	}
	return &longrunningpb.Operation{Name: fmt.Sprintf("operations/testGroup%v", req.Group.GetName()), Done: true, Result: &longrunningpb.Operation_Response{Response: groupPb}}, nil
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()

	fakeService := fakeMCService{groups: make(map[string]*migrationcenterpb.Group)}
	migrationcenterpb.RegisterMigrationCenterServer(srv, &fakeService)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Serve(lis)
		close(serverErr)

	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(ctx, "localhost:9972", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.DialContext err: %v", err)
	}

	mcClient, err := migrationcenter.NewClient(ctx, option.WithGRPCConn(conn))
	if err != nil {
		t.Errorf("Couldn't create client, err: %v", err)
	}

	t.Cleanup(func() {
		lis.Close()
		srv.Stop()
		conn.Close()
		mcClient.Close()

		if err := <-serverErr; err != nil {
			t.Errorf("Server err: %v", err)
		}
	})

	testCases := []struct {
		name           string
		groups         []string
		wantGroups     []*migrationcenterpb.Group
		existingGroups []*migrationcenterpb.Group
		gc             *mcGroupCreator
		ignoreExist    bool
	}{
		{
			name:       "one_group",
			groups:     []string{"chrome"},
			wantGroups: createGroups([]string{"chrome"}, true),
			gc:         &mcGroupCreator{mcutil.ProjectAndLocation{ProjectID: "test", Location: "us-central1"}, mcClient},
		},
		{
			name:       "multiple_groups",
			groups:     []string{"chrome", "gmail", "drive", "MC"},
			wantGroups: createGroupsWithLabel([]string{"chrome", "gmail", "drive", "MC"}),
			gc:         &mcGroupCreator{mcutil.ProjectAndLocation{ProjectID: "test", Location: "us-central1"}, mcClient},
		},
		{
			name:           "one_existing_group_with_mconnect_label",
			groups:         []string{"chrome"},
			wantGroups:     createGroupsWithLabel([]string{"chrome"}),
			existingGroups: createGroupsWithLabel([]string{"chrome"}),
			gc:             &mcGroupCreator{mcutil.ProjectAndLocation{ProjectID: "test", Location: "us-central1"}, mcClient},
			ignoreExist:    true,
		},
		{
			name:           "multiple_existing_groups_with_mconnect_label",
			groups:         []string{"chrome", "gmail", "drive", "MC"},
			wantGroups:     createGroupsWithLabel([]string{"chrome", "gmail", "drive", "MC"}),
			existingGroups: createGroupsWithLabel([]string{"chrome", "gmail", "drive", "MC"}),
			gc:             &mcGroupCreator{mcutil.ProjectAndLocation{ProjectID: "test", Location: "us-central1"}, mcClient},
			ignoreExist:    true,
		},
		{
			name:           "one_existing_group_without_mconnect_label",
			groups:         []string{"chrome"},
			wantGroups:     createGroupsWithLabel([]string{"chrome"}),
			existingGroups: createGroupsWithoutLabel([]string{"chrome"}),
			gc:             &mcGroupCreator{mcutil.ProjectAndLocation{ProjectID: "test", Location: "us-central1"}, mcClient},
			ignoreExist:    true,
		},
		{
			name:           "multiple_existing_groups_without_mconnect_label",
			groups:         []string{"chrome", "gmail", "drive", "MC"},
			wantGroups:     createGroupsWithLabel([]string{"chrome", "gmail", "drive", "MC"}),
			existingGroups: createGroupsWithoutLabel([]string{"chrome", "gmail", "drive", "MC"}),
			gc:             &mcGroupCreator{mcutil.ProjectAndLocation{ProjectID: "test", Location: "us-central1"}, mcClient},
			ignoreExist:    true,
		},
	}

	for _, tc := range testCases {
		fakeService.initGroups(tc.existingGroups)
		t.Run(tc.name, func(t *testing.T) {
			err := tc.gc.create(ctx, tc.groups, tc.ignoreExist)
			if err != nil {
				t.Errorf("Failed creating groups, err: '%v'", err)
			}

			gotGroups := make([]*migrationcenterpb.Group, 0, len(fakeService.groups))
			for _, group := range fakeService.groups {
				gotGroups = append(gotGroups, group)
			}

			opts := cmp.Options{
				cmpopts.EquateEmpty(),
				cmpopts.SortSlices(groupLess),
				cmp.Comparer(equalGroups),
			}
			if diff := cmp.Diff(tc.wantGroups, gotGroups, opts); diff != "" {
				t.Errorf("Diff in groups (-want +got):\n%s", diff)
			}

		})
	}
}

func TestCreateErrors(t *testing.T) {
	ctx := context.Background()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()

	fakeService := fakeMCService{groups: make(map[string]*migrationcenterpb.Group)}
	migrationcenterpb.RegisterMigrationCenterServer(srv, &fakeService)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Serve(lis)
		close(serverErr)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(ctx, "localhost:9972", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.DialContext err: %v", err)
	}

	mcClient, err := migrationcenter.NewClient(ctx, option.WithGRPCConn(conn))
	if err != nil {
		t.Errorf("Couldn't create client, err: %v", err)
	}

	t.Cleanup(func() {
		lis.Close()
		srv.Stop()
		conn.Close()
		mcClient.Close()

		if err := <-serverErr; err != nil {
			t.Errorf("Server err: %v", err)
		}
	})

	testCases := []struct {
		name           string
		groups         []string
		existingGroups []*migrationcenterpb.Group
		gc             *mcGroupCreator
		ignoreExist    bool
		wantError      error
	}{
		{
			name:           "one_group",
			groups:         []string{"chrome"},
			existingGroups: createGroupsWithLabel([]string{"chrome"}),
			gc:             &mcGroupCreator{mcutil.ProjectAndLocation{ProjectID: "test", Location: "us-central1"}, mcClient},
			wantError:      status.Error(codes.AlreadyExists, "already exists"),
		},
	}

	for _, tc := range testCases {
		fakeService.initGroups(tc.existingGroups)
		t.Run(tc.name, func(t *testing.T) {

			err := tc.gc.create(ctx, tc.groups, tc.ignoreExist)
			if gapiutil.IsErrorWithCode(tc.wantError, int(status.Code(err))) {
				t.Errorf("Unexpected error, want: %v, got: %v", tc.wantError, err)
			}

		})
	}

}

func createGroups(groups []string, withLabel bool) []*migrationcenterpb.Group {
	mcGroups := make([]*migrationcenterpb.Group, len(groups))
	labels := make(map[string]string)

	if withLabel {
		labels["mconnect"] = "mconnect"
	}
	for i, group := range groups {
		mcGroups[i] = &migrationcenterpb.Group{
			Name:        group,
			DisplayName: group,
			Description: fmt.Sprintf("%v application group. Created using mconnect.", group),
			Labels:      labels,
		}
	}
	return mcGroups
}

func createGroupsWithLabel(groups []string) []*migrationcenterpb.Group {
	return createGroups(groups, true)
}

func createGroupsWithoutLabel(groups []string) []*migrationcenterpb.Group {
	return createGroups(groups, false)
}

func groupLess(a, b *migrationcenterpb.Group) bool {
	if a.Name != b.Name {
		return a.Name < b.Name
	} else if a.DisplayName != b.DisplayName {
		return a.DisplayName < b.DisplayName
	}
	return a.Description < b.Description
}

func equalGroups(a, b *migrationcenterpb.Group) bool {
	if a == b {
		return true
	}
	return a.Name == b.Name && a.DisplayName == b.DisplayName && a.Description == b.Description && reflect.DeepEqual(a.Labels, b.Labels)
}
