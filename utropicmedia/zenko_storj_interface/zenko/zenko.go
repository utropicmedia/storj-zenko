// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package zenko

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/minio/minio-go"
)

// DEBUG allows more detailed working to be exposed through the terminal.
var DEBUG = false

// ConfigZenko defines the variables and types.
type ConfigZenko struct {
	EndPoint        string `json:"zenkoEndpoint"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
}

// ZenkoReader implements an io.Reader interface
type ZenkoReader struct {
	Client *minio.Client
}

// LoadZenkoProperty reads and parses the JSON file.
// that contain a Zenko instance's property.
// and returns all the properties as an object.
func LoadZenkoProperty(fullFileName string) (ConfigZenko, error) { // fullFileName for fetching Zenko credentials from  given JSON filename.
	var configZenko ConfigZenko
	// Open and read the file
	fileHandle, err := os.Open(fullFileName)
	if err != nil {
		return configZenko, err
	}
	defer fileHandle.Close()

	jsonParser := json.NewDecoder(fileHandle)
	jsonParser.Decode(&configZenko)

	// Display read information.
	fmt.Println("Read Zenko configuration from the ", fullFileName, " file")
	fmt.Println("Zenko End Point\t: ", configZenko.EndPoint)
	return configZenko, nil
}

// ConnectToZenko will connect to a Zenko instance,
// based on the read property from an external file.
// It returns a reference to an io.Reader with Zenko instance information
func ConnectToZenko(fullFileName string) (*ZenkoReader, error) { // fullFileName for fetching Zenko credentials from given JSON filename.
	// Read Zenko instance's properties from an external file.
	configZenko, err := LoadZenkoProperty(fullFileName)

	if err != nil {
		log.Printf("Load Zenko Property Error: %s\n", err)
		return nil, err
	}

	fmt.Println("\nConnecting to Zenko...")
	// Initialize minio client object.
	minioClient, err := minio.New(configZenko.EndPoint, configZenko.AccessKeyID, configZenko.SecretAccessKey, true)
	if err != nil {
		log.Fatal(err)
	}

	// Return Zenko connection client.
	return &ZenkoReader{Client: minioClient}, nil
}
