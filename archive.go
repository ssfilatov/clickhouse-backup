package main

import (
	tarArchive "archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func TarDirs(w io.Writer, dirs ...string) error {
	tw := tarArchive.NewWriter(w)
	defer tw.Close()
	for _, dir := range dirs {
		if err := TarDir(tw, dir); err != nil {
			return err
		}
	}
	return nil
}

func TarDir(tw *tarArchive.Writer, dir string) error {
	return tarDir(tw, dir)
}

type devino struct {
	Dev uint64
	Ino uint64
}

func tarDir(tw *tarArchive.Writer, dir string) (err error) {
	t0 := time.Now()
	nFiles := 0
	hLinks := 0
	defer func() {
		td := time.Since(t0)
		if err == nil {
			log.Printf("added to tarball with: %d files, %d hard links (%v)", nFiles, hLinks, td)
		} else {
			log.Printf("error adding to tarball after %d files, %d hard links, %v: %v", nFiles, hLinks, td, err)
		}
	}()

	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("unable to tar files - %v", err.Error())
	}
	if !fi.IsDir() {
		return fmt.Errorf("data path is not a directory - %v", err.Error())
	}

	seen := make(map[devino]string)

	return filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		header, err := tarArchive.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		filename := strings.TrimPrefix(strings.Replace(file, dir, filepath.Base(dir), -1), string(filepath.Separator))
		header.Name = filename

		st := fi.Sys().(*syscall.Stat_t)
		di := devino{
			Dev: st.Dev,
			Ino: st.Ino,
		}
		orig, ok := seen[di]
		if ok {
			header.Typeflag = tarArchive.TypeLink
			header.Linkname = orig
			header.Size = 0

			err = tw.WriteHeader(header)
			if err != nil {
				return err
			}

			hLinks++
			return nil
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}

		io.Copy(tw, f)

		f.Close()

		seen[di] = filename
		nFiles++

		return nil
	})
	return nil
}
