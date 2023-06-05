// Copyright 2023 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package export provides the actual exporting logic for the bigquery_exporter
package export

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	migrationcenter "cloud.google.com/go/migrationcenter/apiv1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/backoff"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/gapiutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/mcutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/messages"
	exporterschema "github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/schema"
)

var errTableExists = messages.NewError(messages.ErrMsgExportTableExists)

// Params are the parameters for the Export function.
type Params struct {
	ProjectID       string
	Region          string
	TargetProjectID string
	Force           bool
	DatasetID       string
	TablePrefix     string
	Schema          *exporterschema.ExporterSchema
	UserAgentSuffix string
}

func normalizeParams(params *Params) {
	if params.TargetProjectID == "" {
		params.TargetProjectID = params.ProjectID
	}

	if params.Schema == nil {
		params.Schema = &exporterschema.EmbeddedSchema
	}
}

func buildClientOptions(params *Params) []option.ClientOption {
	opts := []option.ClientOption{}
	userAgent := messages.UserAgent
	if params.UserAgentSuffix != "" {
		userAgent += "_" + params.UserAgentSuffix
	}
	opts = append(opts, option.WithUserAgent(userAgent))

	return opts
}

func newExportTask(ctx context.Context, dataset *bigquery.Dataset, params *Params, src mcutil.ObjectSource, tableSuffix string, objectCount uint64) func() error {
	tblName := params.TablePrefix + tableSuffix
	return func() error {
		done := make(chan bool, 1)
		defer close(done)
		go func() {
			const timeout = 5 * time.Second
			timer := time.NewTimer(timeout)
			for {
				select {
				case <-done:
					timer.Stop()
					return
				case <-timer.C:
					if src.ObjectsRead() == objectCount || src.ObjectsRead() == 0 {
						// Don't write progress if we haven't started or just finished
						continue
					}
					fmt.Println(messages.ExportTableInProgress{
						TableName:          tblName,
						RecordsTransferred: src.ObjectsRead(),
						RecordCount:        objectCount,
						BytesTransferred:   src.BytesRead(),
					})
					timer.Reset(timeout)
				}
			}
		}()

		err := exportObjects(ctx, dataset, params, src, tblName)
		if err != nil {
			return fmt.Errorf("export %s: %w", tableSuffix, err)
		}

		done <- true

		fmt.Println(messages.ExportTableComplete{
			TableName:        tblName,
			RecordCount:      src.ObjectsRead(),
			BytesTransferred: src.BytesRead(),
		})

		return nil
	}
}

// MCFactory creates a an MC implementation accorording to params
func MCFactory(ctx context.Context, params *Params) (mcutil.MC, error) {
	svc, err := migrationcenter.NewClient(ctx, buildClientOptions(params)...)
	if err != nil {
		return nil, fmt.Errorf("create migration center client: %w", err)
	}

	return &MCv1{
		client: svc,
		schema: params.Schema,
	}, nil
}

// Export exports migration center data to BigQuery
func Export(params *Params) error {
	normalizeParams(params)
	// The operation never times out, the user can just kill the tool.
	ctx := context.Background()

	path := mcutil.ProjectAndLocation{Project: params.ProjectID, Location: params.Region}
	bq, err := bigquery.NewClient(ctx, params.TargetProjectID, buildClientOptions(params)...)
	if err != nil {
		return fmt.Errorf("create bigquery client: %w", err)
	}
	defer bq.Close()

	fmt.Println(messages.ExportCreatingDataset{DatasetID: params.DatasetID})
	dataset := bq.Dataset(params.DatasetID)
	err = dataset.Create(ctx, &bigquery.DatasetMetadata{
		Name: params.DatasetID,
	})
	err = gapiutil.IgnoreErrorWithCode(err, http.StatusConflict)
	if err != nil {
		return fmt.Errorf("create dataset: %w", err)
	}

	mc, err := MCFactory(ctx, params)
	if err != nil {
		return err
	}
	grp, ctx := errgroup.WithContext(ctx)

	assetCount, err := mc.AssetCount(ctx, path)
	if err != nil {
		return fmt.Errorf("fetch asset count: %w", err)
	}
	assetSource := mc.AssetSource(ctx, path)
	groupSource := mc.GroupSource(ctx, path)
	preferenceSetSource := mc.PreferenceSetSource(ctx, path)
	grp.Go(newExportTask(ctx, dataset, params, groupSource, "groups", 0))
	grp.Go(newExportTask(ctx, dataset, params, assetSource, "assets", uint64(assetCount)))
	grp.Go(newExportTask(ctx, dataset, params, preferenceSetSource, "preference_sets", 0))

	err = grp.Wait()
	if err != nil {
		return err
	}

	fmt.Println(messages.ExportComplete{
		BytesTransferred: assetSource.BytesRead() + groupSource.BytesRead() + preferenceSetSource.BytesRead(),
	})

	return nil
}

type objectIterator[T any] struct {
	listRequest   func(nextPageToken string) ([]*T, string, error)
	objects       []*T
	nextPageToken string
	isInitialized bool
}

func (it *objectIterator[T]) Next() (*T, error) {
	ctx := context.Background()
	if len(it.objects) == 0 {
		if it.nextPageToken == "" && it.isInitialized {
			return nil, iterator.Done
		}
		it.isInitialized = true

		var nextPageToken string
		var objs []*T
		err := backoff.RetryUntil(ctx, gapiutil.DefaultBackoff, func() (bool, error) {
			var err error
			objs, nextPageToken, err = it.listRequest(it.nextPageToken)
			// if there was no error, break
			if err == nil {
				return true, nil
			}
			if gapiutil.IsTransientError(err) {
				return false, nil
			}

			return true, err
		})
		if err != nil {
			return nil, err
		}

		if len(objs) == 0 {
			return nil, iterator.Done
		}

		it.objects = objs
		it.nextPageToken = nextPageToken
	}

	obj := it.objects[0]
	it.objects = it.objects[1:] // pop
	return obj, nil
}

type iterable[T any] interface {
	Next() (T, error)
}

type objectReader[T any] struct {
	schema     bigquery.Schema
	it         iterable[T]
	serializer func(obj T) ([]byte, error)

	buf         []byte
	objectsRead uint64
	bytesRead   uint64
}

func newObjectReader[T any](it iterable[*T], root string, schema bigquery.Schema) *objectReader[*T] {
	return &objectReader[*T]{
		serializer: exporterschema.NewSerializer[*T](root, schema),
		it:         it,
		schema:     schema,
	}
}

func (r *objectReader[T]) Schema() bigquery.Schema {
	return r.schema
}

func (r *objectReader[T]) BytesRead() uint64 {
	return r.bytesRead
}

func (r *objectReader[T]) ObjectsRead() uint64 {
	return r.objectsRead
}

// Read reads the next len(p) bytes from the asset stream.
// The return value n is the number of bytes read.
// If the buffer has no data to return, err is io.EOF (unless len(p) is zero); otherwise it is nil.
func (r *objectReader[T]) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	if len(r.buf) == 0 {
		asset, err := r.it.Next()
		if errors.Is(err, iterator.Done) {
			return 0, io.EOF
		}
		if err != nil {
			return 0, err
		}
		r.buf, err = r.serializer(asset)
		if err != nil {
			return 0, err
		}
		r.objectsRead++
	}

	n := copy(buf, r.buf)
	r.bytesRead += uint64(n)
	r.buf = r.buf[n:]

	return n, nil
}

func newMigrationCenterLoadSource[T any](r *objectReader[T]) bigquery.LoadSource {
	// Creating a full blown bigquery.LoadSource requires a lot of low level big query operations.
	// To save on time we create a ReaderSource and feed it the assets as a json stream.
	src := bigquery.NewReaderSource(r)
	src.Schema = r.Schema()
	src.SourceFormat = bigquery.JSON

	return src
}

func exportObjects(ctx context.Context, dataset *bigquery.Dataset, params *Params, src bigquery.LoadSource, tableName string) error {
	tbl := dataset.Table(tableName)
	_, err := tbl.Metadata(ctx)
	if err != nil && !gapiutil.IsErrorWithCode(err, http.StatusNotFound) {
		return err
	}
	if err == nil && !params.Force {
		return errTableExists
	}

	err = gapiutil.IgnoreErrorWithCode(tbl.Delete(ctx), http.StatusNotFound)
	if err != nil {
		return err
	}

	fmt.Println(messages.ExportingDataToTable{TableName: tableName})
	loader := tbl.LoaderFrom(src)
	loader.WriteDisposition = bigquery.WriteTruncate
	job, err := loader.Run(ctx)
	if err != nil {
		return err
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return err
	}

	err = status.Err()
	if err != nil {
		if len(status.Errors) > 0 {
			var sb strings.Builder
			fmt.Fprintf(&sb, "encountered errors during export:")
			for _, err := range status.Errors {
				fmt.Fprintf(&sb, "\n\t%v", err)
			}

			return errors.New(sb.String())
		}
		return err
	}

	return nil
}
