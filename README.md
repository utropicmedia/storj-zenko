# storj-zenko
### Developed using libuplink version : v0.34.6

## Description

Command line application (on Windows/Linux/Mac) for taking data backup from Zenko to Storj.
Application connects to Zenko server and the souce code for interaction to Storj for cloud storage which is written in Golang.

Zenko is infrastructure software to control Data in Multi-Cloud IT Environments without cloud lock-in and has features such as enabling unified data management from anywhere through a secure cloud portal, providing a single S3 endpoint through which data can be stored, retrieved and searched across any location.

### Features of storj-zenko:
* Connects S3 compatible cloud storages (e.g. Amazon AWS, Azure Blob, Google Cloud Storage, Wasabi) to the Zenko instance for backing up their data to StorJ V3 network.
* Upload any type of data from Zenko to Storj (single or multiple at once) whether it is a folder, document, data file, image, video, etc.
* Download uploaded data from Storj to local machine using "debug" option.


## Initial Set-up
To build from scratch, [install Go](https://golang.org/doc/install#install).

```
$ go get -u github.com/minio/minio-go
$ go get -u github.com/urfave/cli
$ go get -u storj.io/storj/lib/uplink
$ go get -u ./...
```

## Set-up Files
* Create a `zenko_property.json` file, with following contents about a Zenko instance:
    * endpoint :- S3 End point of Zenko Instance
    * accessKeyID :- S3 Access Key ID created in Zenko Instance
    * secretAccessKey :- S3 Secret Access Key created in Zenko Instance


```json
    { 
        "endpoint": "zenkoS3EndPoint-without-http",
        "accessKeyID": "zenkoS3AccessKey",
        "secretAccessKey":"zenkoS3SecretAccessKey"
    }
```

* Create a `storj_config.json` file, with Storj network's configuration information in JSON format:
    * apiKey :- API key created in Storj satellite gui
    * satelliteURL :- Storj Satellite URL
    * encryptionPassphrase :- Storj Encryption Passphrase.
    * bucketName :- Storj Bucket name.
    * uploadPath :- Path on Storj Bucket to store data (optional) or "/"
    * serializedScope:- Serialized Scope Key shared while uploading data used to access bucket without API key
    * disallowReads:- Set true to create serialized scope key with restricted read access
    * disallowWrites:- Set true to create serialized scope key with restricted write access
    * disallowDeletes:- Set true to create serialized scope key with restricted delete access

```json
    { 
        "apikey":     "change-me-to-the-api-key-created-in-satellite-gui",
        "satelliteURL":  "us-central-1.tardigrade.io:7777",
        "bucketName":     "change-me-to-desired-bucket-name",
        "uploadPath": "optionalpath/requiredfilename",
        "encryptionpassphrase": "you'll never guess this",
        "serializedScope": "change-me-to-the-api-key-created-in-encryption-access-apiKey",
        "disallowReads": "true/false-to-disallow-reads",
        "disallowWrites": "true/false-to-disallow-writes",
        "disallowDeletes": "true/false-to-disallow-deletes"
    }
```

* Store both these files in a `config` folder. Filename command-line arguments are optional. Default locations are used.

## Run the command-line tool

* Get help
```
$ storj-zenko -h
```

* Check version
```
$ storj-zenko -v
```

* Read files' data from desired Zenko instance and upload it to given Storj network bucket using Serialized Scope Key.  [note: filename arguments are optional.  default locations are used.]
```
$ storj-zenko store ./config/zenko_property.json ./config/storj_config.json  
```

* Read files' data from desired Zenko instance and upload it to given Storj network bucket API key and EncryptionPassPhrase from storj_config.json and creates an unrestricted shareable Serialized Scope Key.  [note: filename arguments are optional. default locations are used.]
```
$ storj-zenko store ./config/zenko_property.json ./config/storj_config.json key
```

* Read files' data from desired Zenko instance and upload it to given Storj network bucket API key and EncryptionPassPhrase from storj_config.json and creates a restricted shareable Serialized Scope Key.  [note: filename arguments are optional. default locations are used. `restrict` can only be used with `key`]
```
$ storj-zenko store ./config/zenko_property.json ./config/storj_config.json key restrict
```

* Read files' data in `debug` mode from desired Zenko instance and upload it to given Storj network bucket.  [note: filename arguments are optional.  default locations are used. Make sure `debug` folder already exist in project folder.]
```
$ storj-zenko store debug ./config/zenko_property.json ./config/storj_config.json  
```

* Read Zenko instance property from a desired JSON file and display all its files
```
$ storj-zenko parse   
```

* Read Zenko instance property in `debug` mode from a desired JSON file and display all its files
```
$ storj-zenko parse debug 
```

* Read and parse Storj network's configuration, in JSON format, from a desired file and upload a sample object
```
$ storj-zenko test 
```

* Read and parse Storj network's configuration, in JSON format, from a desired file and upload a sample object in `debug` mode
```
$ storj-zenko test debug 
```