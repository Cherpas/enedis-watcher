# enedis-watcher
Retrieve data sent by enedis and send them on a GCP bucket

# Context

This program is designed to decrypt files that are sent by Enedis, a electricity provider in France, from an electric meter via FTP. The files contain important information about your electricity usage and are encrypted for security purposes. Our program allows you to easily decrypt these files and send it to a GCP bucket so that you can access and analyze the data they contain.

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

# F.A.Q

### How to set the environment variable ?

You can do it through bash. Here's an example : 

```bash
export ENEDIS_DECRYPTION_KEY=<DECRYPTION_KEY>
export GOOGLE_APPLICATION_CREDENTIALS=/home/enedis/.secrets/user.json
export ENEDIS_FTP_FOLDER=/home/enedis/ftp_folder
export DECRYPTER_JAR_PATH=/home/enedis/jar/Decrypt.jar
```
