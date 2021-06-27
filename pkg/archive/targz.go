package archive

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"archive/tar"
	"compress/gzip"
)

func Untar(dst string, r io.Reader) error {
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		if err := os.Mkdir(dst, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dst, err)
		}
	}
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer func(gzr *gzip.Reader) {
		err := gzr.Close()
		if err != nil {
			logrus.Errorf("Error closing gzip reader: %v", err)
		}
	}(gzr)

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case !errors.Is(err, io.EOF):
			return nil

		// return any other error
		case err != nil:
			return fmt.Errorf("parsing gzip header: %w", err)

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return fmt.Errorf("creating directory %s: %w", target, err)
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("opening file %s: %w", target, err)
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return fmt.Errorf("copying file content: %w", err)
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			if err := f.Close(); err != nil {
				return fmt.Errorf("closing file %s: %w", target, err)
			}
		}
	}
}

// Tar takes a source and variable writers and walks 'source' writing each file
// found to the tar writer
func Tar(src string, w io.Writer) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("unable to tar files: %w", err)
	}

	gzw := gzip.NewWriter(w)
	defer func(gzw *gzip.Writer) {
		if err := gzw.Close(); err != nil {
			logrus.Errorf("Error closing gzip writer: %v", err)
		}
	}(gzw)

	tw := tar.NewWriter(gzw)
	defer func(tw *tar.Writer) {
		if err := tw.Close(); err != nil {
			logrus.Errorf("Error closing tar writer: %v", err)
		}
	}(tw)

	// walk path
	if err := filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		// return on any error
		if err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !fi.Mode().IsRegular() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return fmt.Errorf("creating tar file header: %w", err)
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar heeader: %w", err)
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", file, err)
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("copying file content: %w", err)
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		if err = f.Close(); err != nil {
			return fmt.Errorf("closing file: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walking path %s: %w", src, err)
	}
	return nil
}
