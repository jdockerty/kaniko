/*
Copyright 2018 Google LLC

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

package buildcontext

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	kConfig "github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// GCS struct for Google Cloud Storage processing
type GCS struct {
	context string
}

func (g *GCS) UnpackTarFromBuildContext() (string, error) {
	bucket, item := util.GetBucketAndItem(g.context)
	return kConfig.BuildContextDir, unpackTarFromGCSBucket(bucket, item, kConfig.BuildContextDir)
}

func UploadToBucket(r io.Reader, dest string) error {
	ctx := context.Background()
	context := strings.SplitAfter(dest, "://")[1]
	bucketName, item := util.GetBucketAndItem(context)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	bucket := client.Bucket(bucketName)
	w := bucket.Object(item).NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

// unpackTarFromGCSBucket unpacks the context.tar.gz file in the given bucket to the given directory
func unpackTarFromGCSBucket(bucketName, item, directory string) error {
	// Get the tar from the bucket
	tarPath, err := getTarFromBucket(bucketName, item, directory)
	if err != nil {
		return err
	}
	logrus.Debug("Unpacking source context tar...")
	if err := util.UnpackCompressedTar(tarPath, directory); err != nil {
		return err
	}
	// Remove the tar so it doesn't interfere with subsequent commands
	logrus.Debugf("Deleting %s", tarPath)
	return os.Remove(tarPath)
}

// getTarFromBucket gets context.tar.gz from the GCS bucket and saves it to the filesystem
// It returns the path to the tar file
func getTarFromBucket(bucketName, item, directory string) (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	bucket := client.Bucket(bucketName)
	// Get the tarfile context.tar.gz from the GCS bucket, and save it to a tar object
	reader, err := bucket.Object(item).NewReader(ctx)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	tarPath := filepath.Join(directory, constants.ContextTar)
	if err := util.CreateFile(tarPath, reader, 0600, 0, 0); err != nil {
		return "", err
	}
	logrus.Debugf("Copied tarball %s from GCS bucket %s to %s", constants.ContextTar, bucketName, tarPath)
	return tarPath, nil
}
