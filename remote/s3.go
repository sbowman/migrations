package remote

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/sbowman/migrations"
)

var (
	// ErrInvalidPath returned if the migration path is in an unexpected,
	// unparseable format.'
	ErrInvalidPath = errors.New("invalid path; expects bucket/migration")

	// ErrNotFound returned if the object or bucket doesn't exist.
	ErrNotFound = errors.New("not found")
)

// S3Reader implements the migrations ReadWrite IO interface to support
// migrations in an S3 bucket.
type S3Reader struct {
	session *session.Session
	service *s3.S3
}

// InitS3 configures the migration tool to use S3 for migration files, rather
// than local disk.  Expects credentials to be in ~/.aws/credentials.
func InitS3(region string) error {
	s3r, err := NewS3Reader(region)
	if err != nil {
		return err
	}

	migrations.IO = s3r

	return nil
}

// PushS3 pushes the migrations defined in the local directory into the S3
// region's bucket.
func PushS3(local, region, bucket string) error {
	s3r, err := NewS3Reader(region)
	if err != nil {
		return err
	}

	return s3r.PushS3(local, bucket)
}

// NewS3Reader constructs a new S3 IO interface for SQL migrations.
func NewS3Reader(region string) (*S3Reader, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	s3rw := &S3Reader{
		session: sess,
		service: s3.New(sess),
	}

	return s3rw, nil
}

// PushS3 will copy the local migration files to the S3 bucket.  If the file
// doesn't exist on S3 or the local timestamp is newer than the S3 timestamp,
// will push the migration to the S3 bucket.
//
// If the bucket doesn't exist, will create it.
func (s3r *S3Reader) PushS3(local, bucket string) error {
	files, err := ioutil.ReadDir(local)
	if err != nil {
		return err
	}

	for _, info := range files {
		if !strings.HasSuffix(info.Name(), ".sql") {
			continue
		}

		if err = s3r.CreateDirectory(bucket); err != nil {
			return err
		}

		// Don't continously update existing migrations
		ts, err := s3r.Exists(bucket, info.Name())
		if err == nil {
			if info.ModTime().Before(ts) || info.ModTime().Equal(ts) {
				continue
			}
		} else if err != ErrNotFound {
			return err
		}

		path := fmt.Sprintf("%s%c%s", local, os.PathSeparator, info.Name())
		doc, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		if err = s3r.WriteMigration(bucket, info.Name(), doc); err != nil {
			return err
		}
	}

	return nil
}

// CreateDirectory creates an S3 bucket for the migrations if not already
// present.
func (s3r *S3Reader) CreateDirectory(bucket string) error {
	_, err := s3r.service.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		return nil
	}

	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() != s3.ErrCodeNoSuchBucket {
			return err
		}
	} else {
		return err
	}

	_, err = s3r.service.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})

	migrations.Log.Infof("Created migrations bucket %s", bucket)

	return err
}

// WriteMigration writes the migration file to S3.  Expects a path for the
// bucket/migration file.
func (s3r *S3Reader) WriteMigration(bucket, filename string, migration []byte) error {
	r := bytes.NewBuffer(migration)

	uploader := s3manager.NewUploader(s3r.session)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   r,
	})

	migrations.Log.Infof("Updated migration %s in bucket %s", filename, bucket)

	return err
}

// Files retrieves the migration names (keys) in the bucket.
func (s3r *S3Reader) Files(bucket string) ([]string, error) {
	resp, err := s3r.service.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, item := range resp.Contents {
		paths = append(paths, *item.Key)
	}

	return paths, nil
}

// Exists checks if the migration file exists in the bucket.  If it does,
// returns the size of the migration.  If not, returns ErrNotFound.
func (s3r *S3Reader) Exists(bucket, migration string) (time.Time, error) {
	resp, err := s3r.service.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(migration),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFound" {
				return time.Time{}, ErrNotFound
			}
		}

		return time.Time{}, err
	}

	return *resp.LastModified, nil
}

// Read the SQL migration from S3.  Expects path to be the bucket/migration
// file.
func (s3r *S3Reader) Read(path string) (io.Reader, error) {
	bucket, filename, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	downloader := s3manager.NewDownloader(s3r.session)

	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(buf.Bytes()), nil
}

// Parse path in the format "<bucket>/<migration file>".
func parsePath(path string) (bucket string, filename string, err error) {
	_, err = fmt.Sscanf(path, fmt.Sprintf("%%s%c%%s", os.PathSeparator), &bucket, &filename)
	if err != nil {
		err = ErrInvalidPath
	}
	return
}
