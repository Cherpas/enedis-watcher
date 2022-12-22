# enedis-watcher
Watch over a folder linked to FTP in order to decrypt it and send it to a GCP Bucket.

# Warning

The current configuration require the use of a JAR in order to decrypt the file sent by Enedis.
This is not optimal as the decryption process should ideally been made inside `main.go`
If you're not satisfied with this solution don't hesitate to send us a merge request :)

# Installation

## Pre requirements
Java 11 must be installed

The followings environment variables must be set : 

- ENEDIS_DECRYPTION_KEY : You're private decryption key sent by Enedis
- GOOGLE_APPLICATION_CREDENTIALS : Path to GCP service account JSON
- ENEDIS_FTP_FOLDER : The folder watched over
- DECRYPTER_JAR_PATH : The path of the Decrypt.jar



## Install the package

```
# go 1.19+
go install github.com/Cherpas/enedis-watcher@latest
```
