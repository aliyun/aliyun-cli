package lib

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// SchemePrefix is the prefix of oss url
const SchemePrefix string = "oss://"

type CloudURLType string

const (
	CloudURLNone    CloudURLType = "none"
	CloudURLService CloudURLType = "service"
	CloudURLBucket  CloudURLType = "bucket"
	CloudURLObject  CloudURLType = "object"
)

// StorageURLer is the interface for all url
type StorageURLer interface {
	IsCloudURL() bool
	IsFileURL() bool
	ToString() string
}

// CloudURL describes oss url
type CloudURL struct {
	urlStr string
	bucket string
	object string
}

// Init is used to create a cloud url from a user input url
func (cu *CloudURL) Init(urlStr, encodingType string) error {
	cu.urlStr = urlStr
	if err := cu.parseBucketObject(encodingType); err != nil {
		return err
	}
	if err := cu.checkBucketObject(encodingType); err != nil {
		return err
	}
	return nil
}

func (cu *CloudURL) parseBucketObject(encodingType string) error {
	var err error
	path := cu.urlStr

	if strings.HasPrefix(strings.ToLower(path), SchemePrefix) {
		path = string(path[len(SchemePrefix):])
	} else {
		// deal with the url: /bucket/object
		if strings.HasPrefix(path, "/") {
			path = string(path[1:])
		}
	}

	sli := strings.SplitN(path, "/", 2)
	cu.bucket = sli[0]
	if len(sli) > 1 {
		cu.object = sli[1]
		if encodingType == URLEncodingType {
			if cu.object, err = url.QueryUnescape(cu.object); err != nil {
				return fmt.Errorf("invalid cloud url: %s, object name is not url encoded, %s", cu.urlStr, err.Error())
			}
		}
	}
	return nil
}

func (cu *CloudURL) checkBucketObject(encodingType string) error {
	if cu.bucket == "" && cu.object != "" {
		return fmt.Errorf("invalid cloud url: %s, miss bucket", cu.urlStr)
	}
	if encodingType == URLEncodingType && cu.bucket != "" && cu.object == "" {
		if bucket, err := url.QueryUnescape(cu.bucket); err == nil && bucket != cu.bucket {
			return fmt.Errorf("invalid cloud url: %s, bucket url do not support --encoding-type option", cu.urlStr)
		}
	}
	return nil
}

func (cu *CloudURL) checkObjectPrefix() error {
	if strings.HasPrefix(cu.object, "/") {
		return fmt.Errorf("invalid cloud url: %s, object name should not begin with \"/\"", cu.urlStr)
	}
	if strings.HasPrefix(cu.object, "\\") {
		return fmt.Errorf("invalid cloud url: %s, object name should not begin with \"\\\"", cu.urlStr)
	}
	return nil
}

func (cu *CloudURL) checkIsObjectURL() error {
	if cu.bucket == "" {
		return fmt.Errorf("invalid cloud url: %s, miss bucket", cu.urlStr)
	}
	if cu.object == "" {
		return fmt.Errorf("invalid cloud url: %s, miss object", cu.urlStr)
	}
	return nil
}

// IsCloudURL shows if the url is a cloud url
func (cu CloudURL) IsCloudURL() bool {
	return true
}

// IsFileURL shows if the url is a file url
func (cu CloudURL) IsFileURL() bool {
	return false
}

// ToString reconstruct url
func (cu CloudURL) ToString() string {
	if cu.object == "" {
		return fmt.Sprintf("%s%s", SchemePrefix, cu.bucket)
	}
	return fmt.Sprintf("%s%s/%s", SchemePrefix, cu.bucket, cu.object)
}

// FileURL describes file url
type FileURL struct {
	urlStr string
}

// Init simulate inheritance, and polymorphism
func (fu *FileURL) Init(urlStr, encodingType string) error {
	if encodingType == URLEncodingType {
		vurl, err := url.QueryUnescape(urlStr)
		if err != nil {
			return fmt.Errorf("invalid cloud url: %s, file name is not url encoded, %s", urlStr, err.Error())
		}
		urlStr = vurl
	}

	if len(urlStr) >= 2 && urlStr[:2] == "~"+string(os.PathSeparator) {
		homeDir := currentHomeDir()
		if homeDir != "" {
			urlStr = strings.Replace(urlStr, "~", homeDir, 1)
		} else {
			return fmt.Errorf("current home dir is empty")
		}
	}
	fu.urlStr = urlStr
	return nil
}

// IsCloudURL simulate inheritance, and polymorphism
func (fu FileURL) IsCloudURL() bool {
	return false
}

// IsFileURL simulate inheritance, and polymorphism
func (fu FileURL) IsFileURL() bool {
	return true
}

// ToString simulate inheritance, and polymorphism
func (fu FileURL) ToString() string {
	return fu.urlStr
}

// StorageURLFromString analysis input url type and build a storage url from the url
func StorageURLFromString(urlStr, encodingType string) (StorageURLer, error) {
	if strings.HasPrefix(strings.ToLower(urlStr), SchemePrefix) {
		var cloudURL CloudURL
		if err := cloudURL.Init(urlStr, encodingType); err != nil {
			return nil, err
		}
		return cloudURL, nil
	}
	var fileURL FileURL
	if err := fileURL.Init(urlStr, encodingType); err != nil {
		return nil, err
	}
	return fileURL, nil
}

// CloudURLFromString get a oss url from url, if url is not a cloud url, return error
func CloudURLFromString(urlStr, encodingType string) (CloudURL, error) {
	storageURL, err := StorageURLFromString(urlStr, encodingType)
	if err != nil {
		return CloudURL{}, err
	}
	if !storageURL.IsCloudURL() {
		return CloudURL{}, fmt.Errorf("invalid cloud url: \"%s\", please make sure the url starts with: \"%s\"", urlStr, SchemePrefix)
	}
	return storageURL.(CloudURL), nil
}

// ObjectURLFromString get a oss url from url, if url is not a cloud url, return error
func ObjectURLFromString(urlStr, encodingType string) (CloudURL, error) {
	cloudURL, err := CloudURLFromString(urlStr, encodingType)
	if err != nil {
		return cloudURL, err
	}
	return cloudURL, cloudURL.checkIsObjectURL()
}

// CloudURLToString format url string from input
func CloudURLToString(bucket string, object string) string {
	cloudURL := CloudURL{
		bucket: bucket,
		object: object,
	}
	return cloudURL.ToString()
}
