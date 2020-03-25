package main

import (
	//Standard Packages

	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"utropicmedia/zenko_storj_interface/storj"
	"utropicmedia/zenko_storj_interface/zenko"

	"github.com/minio/minio-go"
	"github.com/urfave/cli"
)

var gbDEBUG = false

const zenkoConfigFile = "./config/zenko_property.json"
const storjConfigFile = "./config/storj_config.json"

// Create command-line tool to read from CLI.
var app = cli.NewApp()

// SetAppInfo sets information about the command-line application.
func setAppInfo() {
	app.Name = "Storj Zenko Connector"
	app.Usage = "Backup your File from Zenko Orbit to the decentralized Storj network"
	app.Authors = []*cli.Author{{Name: "Satyam Shivam - Utropicmedia", Email: "development@utropicmedia.com"}}
	app.Version = "1.0.0"
}

// helper function to flag debug
func setDebug(debugVal bool) {
	gbDEBUG = true
	storj.DEBUG = debugVal
}

// setCommands sets various command-line options for the app.
func setCommands() {

	app.Commands = []*cli.Command{
		{
			Name:    "parse",
			Aliases: []string{"p"},
			Usage:   "Command to read and parse JSON information about Zenko instance properties and then generate a list of all files with their corresponding paths",
			//\narguments-\n\t  fileName [optional] = provide full file name (with complete path), storing Zenko properties
			// if this fileName is not given, then data is read from ./config/zenko_property.json\n\t
			// example = ./storj-zenko p ./config/zenko_property.json\n",
			Action: func(cliContext *cli.Context) error {
				var fullFileName = zenkoConfigFile
				// process arguments
				if len(cliContext.Args().Slice()) > 0 {
					for i := 0; i < len(cliContext.Args().Slice()); i++ {
						// In case, debug is provided as argument.
						if cliContext.Args().Slice()[i] == "debug" {
							setDebug(true)
						} else {
							fullFileName = cliContext.Args().Slice()[i]
						}
					}
				}

				// Establish connection with Zenko and get io.Reader implementor.
				zenkoReader, err := zenko.ConnectToZenko(fullFileName)
				if err != nil {
					log.Fatalf("Failed to establish connection with Zenko: %s\n", err)
				}

				// Create a done channel to control 'ListObjects' go routine.
				doneCh := make(chan struct{})

				// Indicate to routine to exit cleanly upon return.
				defer close(doneCh)

				isRecursive := true
				// List all buckets from Zenko Orbit.
				buckets, err := zenkoReader.Client.ListBuckets()
				if err != nil {
					log.Fatal(err)
				}

				// Inform about successful connection.
				fmt.Println("Successfully connected to Zenko!")

				for _, bucket := range buckets {
					fmt.Printf("\n\nReading All files from the Zenko Orbit Bucket %s...\n", bucket.Name)
					// ListObjects lists all objects from the specified bucket.
					objectCh := zenkoReader.Client.ListObjects(bucket.Name, "", isRecursive, doneCh)
					for object := range objectCh {
						if object.Err != nil {
							log.Fatal("Object Information Error: ", object.Err)
						}
						fmt.Println(object.Key)
					}
				}

				fmt.Println("\nReading ALL files from the Zenko Orbit Bucket...Complete!")
				return err
			},
		},

		{
			Name:    "test",
			Aliases: []string{"t"},
			Usage:   "Command to read and parse JSON information about Storj network and upload sample data",
			//\n arguments- 1. fileName [optional] = provide full file name (with complete path), storing Storj configuration information
			//if this fileName is not given, then data is read from ./config/storj_config.json
			//example = ./storj-zenko t ./config/storj_config.json\n\n\n",
			Action: func(cliContext *cli.Context) error {

				// Default Storj configuration file name.
				var fullFileName = storjConfigFile
				var foundFirstFileName = false
				var foundSecondFileName = false
				var keyValue string
				var restrict string

				gbDEBUG = false

				// process arguments
				if len(cliContext.Args().Slice()) > 0 {
					for i := 0; i < len(cliContext.Args().Slice()); i++ {

						// Incase, debug is provided as argument.
						if cliContext.Args().Slice()[i] == "debug" {
							setDebug(true)
						} else {
							if !foundFirstFileName {
								fullFileName = cliContext.Args().Slice()[i]
								foundFirstFileName = true
							} else {
								if !foundSecondFileName {

									keyValue = cliContext.Args().Slice()[i]
									foundSecondFileName = true
								} else {
									restrict = cliContext.Args().Slice()[i]
								}
							}
						}
					}
				}
				//Sample data to be uploaded with sample data name
				fileName := "testdata"
				testData := "test"
				data := []byte(testData)

				if gbDEBUG {
					t := time.Now()
					time := t.Format("2006-01-02")
					fileName = "uploaddata_" + time + ".txt"
					err := ioutil.WriteFile(fileName, data, 0644)
					if err != nil {
						fmt.Println("Error while writting to file: ", err)
					}
				}
				var fileNamesDEBUG []string

				// Connect to storj network.
				ctx, uplink, project, bucket, storjConfig, _, errr := storj.ConnectStorjReadUploadData(fullFileName, keyValue, restrict)

				// Upload sample data on storj network.
				fileNamesDEBUG = storj.ConnectUpload(ctx, bucket, data, fileName, fileNamesDEBUG, storjConfig, errr)

				if errr != nil {
					return errr
				}

				// Close storj project.
				storj.CloseProject(uplink, project, bucket)

				fmt.Println("\nUpload \"testdata\" on Storj: Successful!")
				return errr
			},
		},
		{
			Name:    "store",
			Aliases: []string{"s"},
			Usage:   "Command to connect and transfer file(s)/folder(s) from a desired Zenko Orbit account to given Storj Bucket.",
			//\n    arguments-\n      1. fileName [optional] = provide full file name (with complete path),
			// storing zenko properties in JSON format\n   if this fileName is not given,
			// then data is read from ./config/zenko_property.json\n
			// 2. fileName [optional] = provide full file name (with complete path), storing Storj
			// configuration in JSON format\n     if this fileName is not given, then
			// data is read from ./config/storj_config.json\n
			// example = ./storj-zenko s ./config/zenko_property.json ./config/storj_config.json fileName/DirectoryName\n"
			Action: func(cliContext *cli.Context) error {

				// Default configuration file names.
				var fullFileNameStorj = storjConfigFile
				var fullFileNameZenko = zenkoConfigFile
				var keyValue string
				var restrict string
				var fileNamesDEBUG []string

				// process arguments - Reading file names from the command line.
				var foundFirstFileName = false
				var foundSecondFileName = false
				var foundThirdFileName = false

				if len(cliContext.Args().Slice()) > 0 {
					for i := 0; i < len(cliContext.Args().Slice()); i++ {
						// Incase debug is provided as argument.
						if cliContext.Args().Slice()[i] == "debug" {
							setDebug(true)
						} else {
							if !foundFirstFileName {
								fullFileNameZenko = cliContext.Args().Slice()[i]
								foundFirstFileName = true
							} else {
								if !foundSecondFileName {
									fullFileNameStorj = cliContext.Args().Slice()[i]
									foundSecondFileName = true
								} else {
									if !foundThirdFileName {
										keyValue = cliContext.Args().Slice()[i]
										foundThirdFileName = true
									} else {
										restrict = cliContext.Args().Slice()[i]
									}
								}
							}
						}
					}
				}

				// Establish connection with Zenko and get io.Reader implementor.
				zenkoReader, err := zenko.ConnectToZenko(fullFileNameZenko)
				if err != nil {
					log.Fatalf("Failed to establish connection with Zenko: %s\n", err)
				}

				// Create a done channel to control 'ListObjects' go routine.
				doneCh := make(chan struct{})

				// Indicate to our routine to exit cleanly upon return.
				defer close(doneCh)

				isRecursive := true

				// List of all buckets from Zenko orbit.
				buckets, err := zenkoReader.Client.ListBuckets()
				if err != nil {
					log.Fatal("List Bucket Error:", err)
				}

				// Inform about successful connection.
				fmt.Println("Successfully connected to Zenko!")

				// Connect to storj network and returns context, uplink, project, bucket and storj configuration.
				ctx, uplink, project, bucket, storjConfig, scope, errr := storj.ConnectStorjReadUploadData(fullFileNameStorj, keyValue, restrict)
				if errr != nil {
					log.Fatal(err)
				}

				storePath := make([]string, 0)
				storeExt := make([]string, 0)
				t := time.Now()
				timeNow := t.Format("2006-01-02_15_04_05")
				for _, zenkoBucket := range buckets {
					// ListObjects lists all objects from the specified bucket.
					objectCh := zenkoReader.Client.ListObjects(zenkoBucket.Name, "", isRecursive, doneCh)
					for object := range objectCh {
						if object.Err != nil {
							log.Fatal("Object Information Error", object.Err)
						}

						fmt.Println("\nReading content from the file :", object.Key)
						var temp int64 = 0
						var fileExtension string
						var zenkoPath string
						i := 0
						for temp < object.Size {
							// GetObject function returns seekable, readable object.
							objectReader, err := zenkoReader.Client.GetObject(zenkoBucket.Name, object.Key, minio.GetObjectOptions{})

							if err != nil {
								log.Fatal(err)
							}

							section := io.NewSectionReader(objectReader, temp,32*1024)
							bytes, err := ioutil.ReadAll(section)
							if err != nil {
								log.Fatal(err)
							}

							var path string
							if filepath.Dir(object.Key) == "." {
								path = ""
							} else {
								path = filepath.Dir(object.Key) + "/"
							}
							file := filepath.Base(object.Key)
							split := strings.Split(file, ".")
							fileExtension = split[len(split)-1]
							if len(split) > 2 {
								fileExtension = split[len(split)-2] + "." + split[len(split)-1]
							}
							lastFileName := split[0]
							for i := 1; i < len(split)-2; i++ {
								lastFileName = lastFileName + split[i] + "."
								if i == len(split)-3 {
									lastFileName = lastFileName + split[i]
								}

							}
							zenkoPath = zenkoBucket.Name + "_" + timeNow + "/" + path + lastFileName
							zenkoFilePath := zenkoPath + "/" + strconv.Itoa(i) + "." + fileExtension
							i++
							// Upload Zenko object on storj Network with file name.
							storj.ConnectUpload(ctx, bucket, bytes, zenkoFilePath, fileNamesDEBUG, storjConfig, errr)
							if errr != nil {
								log.Fatal(errr)
							}
							temp = temp + int64(len(bytes))
						}
						storeExt = append(storeExt, fileExtension)
						storePath = append(storePath, zenkoPath)
					}
				}
				// Debug the StorJ data.
				storj.Debug(ctx, bucket, storjConfig.UploadPath, storePath, storeExt)

				// Close the StorJ project.
				storj.CloseProject(uplink, project, bucket)
				fmt.Println(" ")
				if keyValue == "key" {
					if restrict == "restrict" {
						fmt.Println("Restricted Serialized Scope Key: ", scope)
						fmt.Println(" ")
					} else {
						fmt.Println("Serialized Scope Key: ", scope)
						fmt.Println(" ")
					}
				}

				return err
			},
		},
	}
}

func main() {
	// Show application's information on screen
	setAppInfo()

	// Get command entered by user on cli
	setCommands()

	// Get detailed information for debugging
	setDebug(false)

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("app.Run: %s", err)
	}
}
