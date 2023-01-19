package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/fsnotify/fsnotify"
)

func uploadFileToBucket(bucketName, folderName, filePath string) error {
	// Create a client
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	defer client.Close()
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}

	// Open the file to be uploaded
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("os.Open: %v", err)
	}
	defer f.Close()

	// Create a bucket instance
	bkt := client.Bucket(bucketName)

	// Create a new object
	obj := bkt.Object(fmt.Sprintf("%s/%s", folderName, filepath.Base(filePath)))

	// Create a writer to the object
	w := obj.NewWriter(ctx)

	// Copy the file contents to the object
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}

	log.Printf("File %s uploaded to bucket %s in folder %s\n", filePath, bucketName, folderName)
	return nil
}

func executeDecrypter(jarPath string, completeFileName string, decryptionKey string) (string, error) {

	outputFilePath := "/tmp/" + filepath.Base(completeFileName) + "_DECRYPTED.zip"
	// Use the exec.Command function to create a new command for executing the JAR file.
	cmd := exec.Command("java", "-jar", jarPath, decryptionKey, completeFileName, outputFilePath)

	// Execute the command and return any error that occurs.
	return outputFilePath, cmd.Run()
}

// By default the ZIP file returned by the decrypted is corrupted (the EOCDR is invalid)
// This function repair the ZIP archive using the `zip -FF` shell command
func repairZip(damagedZipFilePath string) (string, error) {
	repairedFilePath := filepath.Dir(damagedZipFilePath) + "/" + "repaired_" + filepath.Base(damagedZipFilePath)

	// Create the command to repair the ZIP file
	// The command need to be answered by a 'y\n' to confirm it's a single-disk archive
	cmd := exec.Command("zip", "-FF", damagedZipFilePath, "--out", repairedFilePath)

	// Create a pipe for the standard input of the zip command
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	// Write the "y" response to the pipe
	_, err = stdin.Write([]byte("y\n"))
	if err != nil {
		panic(err)
	}

	// Close the pipe
	err = stdin.Close()
	if err != nil {
		panic(err)
	}

	// Execute the command and retrieve error
	err = cmd.Run()

	if err != nil {
		return "", err
	}

	// Remove the old corrupted archive
	os.Remove(damagedZipFilePath)

	return repairedFilePath, nil

}

// Extract the ZIP archive into a folder and return it's folder
func extractZip(zipPath string) (string, error) {
	// Open the zip file
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer zipReader.Close()

	// Create a new folder on the same path as the zip file
	dirPath := filepath.Dir(zipPath)
	folderPath := filepath.Join(dirPath, filepath.Base(zipPath)+"_extracted")
	err = os.Mkdir(folderPath, os.ModePerm)
	if err != nil {
		return "", err
	}

	// Extract the files from the zip archive
	for _, file := range zipReader.File {
		// Open the file in the zip archive
		zipFile, err := file.Open()
		if err != nil {
			return "", err
		}
		defer zipFile.Close()

		// Create a new file in the new folder
		path := filepath.Join(folderPath, file.Name)
		newFile, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer newFile.Close()

		// Copy the file from the zip archive to the new file
		_, err = io.Copy(newFile, zipFile)
		if err != nil {
			return "", err
		}
	}

	// Remove the archive for cleaning
	os.Remove(zipPath)

	return folderPath, nil
}

// Upload the content of the unzipped folder to the GCP bucket
func uploadFolderContent(folderPath string,outputBucket string) error {
	// Read the contents of the folder
	contents, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return err
	}

	// Iterate over the elements in the folder
	for _, element := range contents {
		// Get the absolute path of the element
		elementPath, err := filepath.Abs(filepath.Join(folderPath, element.Name()))
		if err != nil {
			return err
		}

		// Execute the upload function on each element
		if err := uploadFileToBucket(outputBucket, "decrypted_xml", elementPath); err != nil {
			return err
		}
	}

	return nil
}

func uploadFolderAnyway(folderPath string,outputBucket string) error {
	// Read the contents of the folder
	contents, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return err
	}

	// Iterate over the elements in the folder
	for _, element := range contents {
		// Get the absolute path of the element
		elementPath, err := filepath.Abs(filepath.Join(folderPath, element.Name()))
		if err != nil {
			return err
		}

		// Execute the upload function on each element
		if err := uploadFileToBucket(outputBucket, "raw_enedis", elementPath); err != nil {
			return err
		}
	}

	return nil
}

func main() {

	// Look for the env variable and exit if they're not any
	decryptionKey, exists := os.LookupEnv("ENEDIS_DECRYPTION_KEY")
	if !exists {
		log.Fatalln("The ENEDIS_DECRYPTION_KEY environment variable must be set")
	}

	if _, exists := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); !exists {
		log.Fatalln("The GOOGLE_CREDENTIALS_FILEPATH environment variable must be set")
	}

	ftpFolder, exists := os.LookupEnv("ENEDIS_FTP_FOLDER")
	if !exists {
		log.Fatalln("The ENEDIS_FTP_FOLDER environment variable must be set")
	}

	jarPath, exists := os.LookupEnv("DECRYPTER_JAR_PATH")
	if !exists {
		log.Fatalln("The DECRYPTER_JAR_PATH environment variable must be set")
	}

	outputBucket, exists := os.LookupEnv("OUTPUT_BUCKET")
	if !exists {
		log.Fatalln("The OUTPUT_BUCKET environment variable must be set")
	}



	// Create a new fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Add the folder to be watched
	err = watcher.Add(ftpFolder)
	if err != nil {
		log.Fatal(err)
	}

	// Set up a channel to receive notifications when the folder is updated
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Fatalf("Error when receiving data from channel")
				}

				// Check if the event is a file being created
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("File added:", event.Name)

					uploadFolderAnyway(event.Name,outputBucket)

					corruptedZipFilePath, err := executeDecrypter(jarPath, event.Name, decryptionKey)

					fmt.Println(corruptedZipFilePath)

					if err != nil {
						log.Println("Error while decrypting "+event.Name+" : ", err)
						continue
					}
					repairedZipFilePath, err := repairZip(corruptedZipFilePath)

					if err != nil {
						log.Println("Error while repairing the archive "+corruptedZipFilePath+" : ", err)
						continue
					}

					folderFilePath, err := extractZip(repairedZipFilePath)

					if err != nil {
						log.Println("Error while extracting the zip " + repairedZipFilePath + " : " + err.Error())
						continue
					}

					uploadFolderContent(folderFilePath,outputBucket)

					// Cleaning
					os.Remove(event.Name)
					os.Remove(corruptedZipFilePath)
					os.Remove(repairedZipFilePath)
					os.RemoveAll(folderFilePath)

				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Fatal("Error:", err)
			}
		}
	}()

	fmt.Println("Server started")

	// Run the watcher in the background
	<-done
}
