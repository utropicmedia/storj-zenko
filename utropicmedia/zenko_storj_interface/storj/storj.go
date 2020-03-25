// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package storj

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"storj.io/storj/lib/uplink"
	"storj.io/storj/pkg/macaroon"
)

// DEBUG allows more detailed working to be exposed through the terminal.
var DEBUG = false

// ConfigStorj depicts keys to search for within the storj_config.json file.
type ConfigStorj struct {
	APIKey               string `json:"apiKey"`
	Satellite            string `json:"satelliteURL"`
	Bucket               string `json:"bucketName"`
	UploadPath           string `json:"uploadPath"`
	EncryptionPassphrase string `json:"encryptionPassphrase"`
	SerializedScope      string `json:"serializedScope"`
	DisallowReads        string `json:"disallowReads"`
	DisallowWrites       string `json:"disallowWrites"`
	DisallowDeletes      string `json:"disallowDeletes"`
}

// LoadStorjConfiguration reads and parses the JSON file that contains Storj configuration's information.
func LoadStorjConfiguration(fullFileName string) (ConfigStorj, error) { // fullFileName for fetching Storj V3 credentials from given JSON filename.

	var configStorj ConfigStorj

	fileHandle, err := os.Open(fullFileName)
	if err != nil {
		return configStorj, err
	}

	defer fileHandle.Close()

	jsonParser := json.NewDecoder(fileHandle)
	jsonParser.Decode(&configStorj)

	return configStorj, nil
}

// ConnectStorjReadUploadData reads Storj configuration from given json file,
// connects to the desired Storj network.
// It then reads data property from an external config file.
func ConnectStorjReadUploadData(fullFileName string, keyValue string, restrict string) (context.Context, *uplink.Uplink, *uplink.Project, *uplink.Bucket, ConfigStorj, string, error) {
	// fullFileName for fetching storj V3 credentials from  given JSON filename
	var scope string
	configStorj, err := LoadStorjConfiguration(fullFileName)
	if err != nil {
		fmt.Printf("LoadStorjConfiguration: %s", err)
	}

	// Display the read information.
	fmt.Println("\nReading Storj configuration from file: ", fullFileName)
	fmt.Println("API Key\t\t\t: ", configStorj.APIKey)
	fmt.Println("Satellite\t	: ", configStorj.Satellite)
	fmt.Println("Bucket	\t	: ", configStorj.Bucket)
	fmt.Println("Upload Path\t\t: ", configStorj.UploadPath)
	fmt.Println("Serialized Scope Key\t: ", configStorj.SerializedScope)
	fmt.Println("\nCreating New Uplink...")

	var cfg uplink.Config

	// Configure the user agent
	cfg.Volatile.UserAgent = "Zenko"

	ctx := context.Background()

	uplinkstorj, err := uplink.NewUplink(ctx, &cfg)
	if err != nil {
		uplinkstorj.Close()
		log.Fatal("Could not create new Uplink object:", err)
	}

	var serializedScope string

	if keyValue == "key" {
		fmt.Println("Parsing the API key...")
		key, err := uplink.ParseAPIKey(configStorj.APIKey)
		if err != nil {
			uplinkstorj.Close()
			log.Fatal("Could not parse API key:", err)
		}

		if DEBUG {
			fmt.Println("API key \t :", configStorj.APIKey)
			fmt.Println("Serialized API key :", key.Serialize())
		}

		fmt.Println("Opening Project...")
		proj, err := uplinkstorj.OpenProject(ctx, configStorj.Satellite, key)

		if err != nil {
			CloseProject(uplinkstorj, proj, nil)
			log.Fatal("Could not open project:", err)
		}

		// Creating an encryption key from encryption passphrase.
		if DEBUG {
			fmt.Println("\nGetting encryption key from pass phrase...")
		}

		encryptionKey, err := proj.SaltedKeyFromPassphrase(ctx, configStorj.EncryptionPassphrase)
		if err != nil {
			CloseProject(uplinkstorj, proj, nil)
			log.Fatal("Could not create encryption key:", err)
		}

		// Creating an encryption context.
		access := uplink.NewEncryptionAccessWithDefaultKey(*encryptionKey)
		if DEBUG {
			fmt.Println("Encryption access \t:", configStorj.EncryptionPassphrase)
		}
		// Serializing the parsed access, so as to compare with the original key.
		serializedAccess, err := access.Serialize()
		if err != nil {
			CloseProject(uplinkstorj, proj, nil)
			log.Fatal("Error Serialized key : ", err)

		}
		if DEBUG {
			fmt.Println("Serialized access key\t:", serializedAccess)
		}

		// Load the existing encryption access context
		accessParse, err := uplink.ParseEncryptionAccess(serializedAccess)
		if err != nil {
			log.Fatal(err)
		}

		if restrict == "restrict" {
			disallowRead, _ := strconv.ParseBool(configStorj.DisallowReads)
			disallowWrite, _ := strconv.ParseBool(configStorj.DisallowWrites)
			disallowDelete, _ := strconv.ParseBool(configStorj.DisallowDeletes)
			userAPIKey, err := key.Restrict(macaroon.Caveat{
				DisallowReads:   disallowRead,
				DisallowWrites:  disallowWrite,
				DisallowDeletes: disallowDelete,
			})
			if err != nil {
				log.Fatal(err)
			}

			userAPIKey, userAccess, err := accessParse.Restrict(userAPIKey,
				uplink.EncryptionRestriction{
					Bucket:     configStorj.Bucket,
					PathPrefix: configStorj.UploadPath,
				},
			)
			if err != nil {
				log.Fatal(err)
			}
			userRestrictScope := &uplink.Scope{
				SatelliteAddr:    configStorj.Satellite,
				APIKey:           userAPIKey,
				EncryptionAccess: userAccess,
			}
			serializedRestrictScope, err := userRestrictScope.Serialize()
			if err != nil {
				log.Fatal(err)
			}
			scope = serializedRestrictScope
		}

		userScope := &uplink.Scope{
			SatelliteAddr:    configStorj.Satellite,
			APIKey:           key,
			EncryptionAccess: access,
		}
		serializedScope, err = userScope.Serialize()
		if err != nil {
			log.Fatal(err)
		}
		if restrict == "" {
			scope = serializedScope
		}

		proj.Close()
		uplinkstorj.Close()
	} else {
		serializedScope = configStorj.SerializedScope

	}
	parsedScope, err := uplink.ParseScope(serializedScope)
	if err != nil {
		log.Fatal(err)
	}

	uplinkstorj, err = uplink.NewUplink(ctx, &cfg)
	if err != nil {
		log.Fatal("Could not create new Uplink object:", err)
	}
	proj, err := uplinkstorj.OpenProject(ctx, parsedScope.SatelliteAddr, parsedScope.APIKey)
	if err != nil {
		CloseProject(uplinkstorj, proj, nil)
		log.Fatal("Could not open project:", err)
	}

	fmt.Println("Opening Bucket\t\t: ", configStorj.Bucket)

	// Open up the desired Bucket within the Project.
	bucket, err := proj.OpenBucket(ctx, configStorj.Bucket, parsedScope.EncryptionAccess)
	if err != nil {
		fmt.Println("Could not open bucket", configStorj.Bucket, ":", err)
		fmt.Println("Trying to create new bucket....")
		_, err1 := proj.CreateBucket(ctx, configStorj.Bucket, nil)
		if err1 != nil {
			CloseProject(uplinkstorj, proj, bucket)
			fmt.Printf("Could not create bucket %q:", configStorj.Bucket)
			log.Fatal(err1)
		} else {
			fmt.Println("Created Bucket", configStorj.Bucket)
		}
		fmt.Println("Opening created Bucket: ", configStorj.Bucket)
		bucket, err = proj.OpenBucket(ctx, configStorj.Bucket, parsedScope.EncryptionAccess)
		if err != nil {
			fmt.Printf("Could not open bucket %q: %s", configStorj.Bucket, err)
		}
	}
	return ctx, uplinkstorj, proj, bucket, configStorj, scope, err
}

// ConnectUpload uploads the data to storj network.
// Read data using io.Reader and upload it to Storj.
func ConnectUpload(ctx context.Context, bucket *uplink.Bucket, data []byte, filename string, fileNamesDEBUG []string, configStorj ConfigStorj, err error) []string {

	checkSlash := configStorj.UploadPath[len(configStorj.UploadPath)-1:]
	if checkSlash != "/" {
		configStorj.UploadPath = configStorj.UploadPath + "/"
	}

	fmt.Println("\nUpload Object Path: ", configStorj.UploadPath+filename)

	readerBytes := bytes.NewReader(data)
	readerIO := io.Reader(readerBytes)
	// Upload the data on storj.
	err = bucket.UploadObject(ctx, configStorj.UploadPath+filename, readerIO, nil)
	if err != nil {
		log.Fatal("Could not upload:", err)
	}
	if DEBUG {
		fileNamesDEBUG = append(fileNamesDEBUG, filename)
	}

	fmt.Println("Uploading object to Storj bucket: Completed!")

	return fileNamesDEBUG
}

// Debug function downloads the data from StorJ bucket to verify data from Zenko is uploaded successfully.
func Debug(ctx context.Context, bucket *uplink.Bucket, uploadPath string, fileName []string, fileExt []string) {
	if DEBUG {

		checkSlash := uploadPath[len(uploadPath)-1:]
		if checkSlash != "/" {
			uploadPath = uploadPath + "/"
		}
		for i := 0; i < len(fileName); i++ {
			fmt.Printf("\n\nDownloading Object %s from bucket : Initiated...\n", uploadPath+fileName[i]+"/")
			list, err := bucket.ListObjects(ctx, &uplink.ListOptions{
				Direction: 2,
				Cursor:    "",
				Prefix:    uploadPath + fileName[i] + "/",
				Recursive: false,
			})
			if err != nil {
				log.Fatal(err)
			}
			for j := 0; j < len(list.Items); j++ {
				//for _, object := range list.Items {
				strm, err := bucket.Download(ctx, uploadPath+fileName[i]+"/"+strconv.Itoa(j)+"."+fileExt[i])
				if err != nil {
					fmt.Printf("Could not open object at %q: %v", strconv.Itoa(j)+"."+fileExt[i], err)
				}
				defer strm.Close()
				// Read everything from the stream.
				receivedContents, err := ioutil.ReadAll(strm)
				if err != nil {
					fmt.Printf("Could not read object: %v", err)
				}
				filePath := filepath.Dir(fileName[i])

				_ = os.MkdirAll("debug/"+uploadPath+filePath+"/", 0755) //Make directory on system
				var fileNameDownload = filepath.Join("debug", uploadPath+fileName[i]+"."+fileExt[i])
				f, err := os.OpenFile(fileNameDownload, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
				_, err = f.Write(receivedContents) //Append the bytes to the file created
				if err != nil {
					fmt.Println(err)
				}

				fmt.Printf("%d bytes of Object from bucket!\n", len(receivedContents))
			}
		}
	}
}

// CloseProject closes bucket, project and uplink.
func CloseProject(uplink *uplink.Uplink, proj *uplink.Project, bucket *uplink.Bucket) {
	if bucket != nil {
		bucket.Close()
	}
	if proj != nil {
		proj.Close()
	}
	if uplink != nil {
		uplink.Close()
	}
}
