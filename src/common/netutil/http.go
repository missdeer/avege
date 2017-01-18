package netutil

import (
	"io"
	"net/http"
	"os"

	"common"
)

func DownloadRemoteContent(remoteLink string) (io.ReadCloser, error) {
	response, err := http.Get(remoteLink)
	if err != nil {
		common.Error("Error while downloading", remoteLink, err)
		return nil, err
	}

	return response.Body, nil
}

func DownloadRemoteFile(remoteLink string, saveToFile string) error {
	response, err := http.Get(remoteLink)
	if err != nil {
		common.Error("Error while downloading", remoteLink, err)
		return err
	}
	defer response.Body.Close()

	output, err := os.Create(saveToFile)
	if err != nil {
		common.Error("Error while creating", saveToFile, err)
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, response.Body); err != nil {
		common.Error("Error while reading response", remoteLink, err)
		return err
	}

	return nil
}
