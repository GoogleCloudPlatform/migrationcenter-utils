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

// Package parser provides an implementation for a parser which parses CAST files.
package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/messages"
)

const (
	expectedHeader = "application"
)

// Groups receives a path to a CAST file which it parses and returns the application names it found.
func Groups(path string) ([]string, error) {

	if err := ValidFileFormat(path); err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	appInd, err := appIndex(headers)
	if err != nil {
		return nil, err
	}

	appsSet := make(map[string]int)
	applications := []string{}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading record: %v\n", err)
			continue
		}

		application := strings.Split(record[appInd], ",")[0]
		if _, ok := appsSet[application]; !ok {
			appsSet[application] = 0
			applications = append(applications, application)

		}
	}

	return applications, nil
}

func appIndex(headers []string) (int, error) {
	for i, header := range headers {
		if strings.ToLower(header) == expectedHeader {
			return i, nil
		}
	}
	return -1, fmt.Errorf("couldn't find the application header")
}

// ValidFileFormat returns an error if the file format doesn't match the CAST allowed file format.
func ValidFileFormat(path string) error {
	// Verifying the CAST file extension is correct.
	extension := filepath.Ext(path)
	if !(extension == ".csv" || extension == ".txt") {
		return messages.WrongFileFormatError{FileFormat: extension}.Error()
	}
	return nil
}
