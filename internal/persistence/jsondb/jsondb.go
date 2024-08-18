// Copyright (C) 2024 The Daguflow/Dagu Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package jsondb

import (
	"bufio"
	// nolint: gosec
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/daguflow/dagu/internal/persistence"
	"github.com/daguflow/dagu/internal/persistence/filecache"

	"github.com/daguflow/dagu/internal/persistence/model"

	"github.com/daguflow/dagu/internal/util"
)

var _ persistence.HistoryStore = (*JSONDB)(nil)

// JSONDB manages dags status files in local.
type JSONDB struct {
	location          string
	writer            *writer
	cache             *filecache.Cache[*model.Status]
	latestStatusToday bool
}

var (
	// errors
	errRequestIDNotFound  = errors.New("request ID not found")
	errCreateNewDirectory = errors.New("failed to create new directory")
	errDAGFileEmpty       = errors.New("dagFile is empty")
)

const defaultCacheSize = 300

// New creates a new JSONDB with default configuration.
func New(location string, latestStatusToday bool) *JSONDB {
	s := &JSONDB{
		location: location,
		cache: filecache.New[*model.Status](
			defaultCacheSize, time.Hour*3,
		),
		latestStatusToday: latestStatusToday,
	}
	s.cache.StartEviction()
	return s
}

func (s *JSONDB) Update(dagFile, requestID string, status *model.Status) error {
	f, err := s.FindByRequestID(dagFile, requestID)
	if err != nil {
		return err
	}
	w := &writer{target: f.File}
	if err := w.open(); err != nil {
		return err
	}
	defer func() {
		s.cache.Invalidate(f.File)
		_ = w.close()
	}()
	return w.write(status)
}

func (s *JSONDB) Open(dagFile string, t time.Time, requestID string) error {
	writer, _, err := s.newWriter(dagFile, t, requestID)
	if err != nil {
		return err
	}
	if err := writer.open(); err != nil {
		return err
	}
	s.writer = writer
	return nil
}

func (s *JSONDB) Write(status *model.Status) error {
	return s.writer.write(status)
}

func (s *JSONDB) Close() error {
	if s.writer == nil {
		return nil
	}
	defer func() {
		_ = s.writer.close()
		s.writer = nil
	}()
	if err := s.Compact(
		s.writer.dagFile, s.writer.target,
	); err != nil {
		return err
	}
	s.cache.Invalidate(s.writer.target)
	return s.writer.close()
}

// NewWriter creates a new writer for a status.
func (s *JSONDB) newWriter(
	dagFile string, t time.Time, requestID string,
) (*writer, string, error) {
	f, err := s.newFile(dagFile, t, requestID)
	if err != nil {
		return nil, "", err
	}
	w := &writer{target: f, dagFile: dagFile}
	return w, f, nil
}

// ReadStatusRecent returns recent n status
func (s *JSONDB) ReadStatusRecent(
	dagFile string, n int,
) []*model.StatusFile {
	var ret []*model.StatusFile
	files := s.latest(s.globPattern(dagFile), n)
	for _, file := range files {
		status, err := s.cache.LoadLatest(
			file,
			func() (*model.Status, error) {
				return ParseFile(file)
			},
		)
		if err != nil {
			continue
		}
		ret = append(ret, &model.StatusFile{
			File:   file,
			Status: status,
		})
	}
	return ret
}

// ReadStatusToday returns a list of status files.
func (s *JSONDB) ReadStatusToday(dagFile string) (*model.Status, error) {
	// TODO: let's fix below not to use config here
	file, err := s.latestToday(dagFile, time.Now(), s.latestStatusToday)
	if err != nil {
		return nil, err
	}
	return s.cache.LoadLatest(file, func() (*model.Status, error) {
		return ParseFile(file)
	})
}

// FindByRequestID finds a status file by request ID
func (s *JSONDB) FindByRequestID(
	dagFile string, requestID string,
) (*model.StatusFile, error) {
	if requestID == "" {
		return nil, errRequestIDNotFound
	}
	matches, err := filepath.Glob(s.globPattern(dagFile))
	if len(matches) > 0 || err == nil {
		sort.Slice(matches, func(i, j int) bool {
			return strings.Compare(matches[i], matches[j]) >= 0
		})
		for _, f := range matches {
			status, err := ParseFile(f)
			if err != nil {
				log.Printf("parsing failed %s : %s", f, err)
				continue
			}
			if status != nil && status.RequestID == requestID {
				return &model.StatusFile{
					File:   f,
					Status: status,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("%w : %s", persistence.ErrRequestIDNotFound, requestID)
}

// RemoveAll removes all files in a directory.
func (s *JSONDB) RemoveAll(dagFile string) error {
	return s.RemoveOld(dagFile, 0)
}

// RemoveOld removes old files.
func (s *JSONDB) RemoveOld(dagFile string, retentionDays int) error {
	var lastErr error
	if retentionDays >= 0 {
		matches, _ := filepath.Glob(s.globPattern(dagFile))
		ot := time.Now().AddDate(0, 0, -1*retentionDays)
		for _, m := range matches {
			info, err := os.Stat(m)
			if err == nil {
				if info.ModTime().Before(ot) {
					lastErr = os.Remove(m)
				}
			}
		}
	}
	return lastErr
}

// Compact creates a new file with only the latest data and removes old data.
func (*JSONDB) Compact(_, original string) error {
	status, err := ParseFile(original)
	if err != nil {
		return err
	}

	newFile := fmt.Sprintf("%s_c.dat",
		strings.TrimSuffix(filepath.Base(original), filepath.Ext(original)))
	f := filepath.Join(filepath.Dir(original), newFile)
	w := &writer{target: f}
	if err := w.open(); err != nil {
		return err
	}
	defer func() {
		_ = w.close()
	}()

	if err := w.write(status); err != nil {
		if err := os.Remove(f); err != nil {
			log.Printf("failed to remove %s : %s", f, err.Error())
		}
		return err
	}

	return os.Remove(original)
}

func (*JSONDB) exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func (s *JSONDB) Rename(oldID, newID string) error {
	// This is needed to ensure backward compatibility.
	on := util.AddYamlExtension(oldID)
	nn := util.AddYamlExtension(newID)

	oldDir := s.getDirectory(on, prefix(on))
	newDir := s.getDirectory(nn, prefix(nn))
	if !s.exists(oldDir) {
		// Nothing to do
		return nil
	}
	if !s.exists(newDir) {
		if err := os.MkdirAll(newDir, 0755); err != nil {
			return fmt.Errorf(
				"%w: %s : %s", errCreateNewDirectory, newDir, err.Error(),
			)
		}
	}
	matches, err := filepath.Glob(s.globPattern(on))
	if err != nil {
		return err
	}
	oldPrefix := filepath.Base(s.prefixWithDirectory(on))
	newPrefix := filepath.Base(s.prefixWithDirectory(nn))
	for _, m := range matches {
		base := filepath.Base(m)
		f := strings.Replace(base, oldPrefix, newPrefix, 1)
		_ = os.Rename(m, filepath.Join(newDir, f))
	}
	if files, _ := os.ReadDir(oldDir); len(files) == 0 {
		_ = os.Remove(oldDir)
	}
	return nil
}

func (s *JSONDB) getDirectory(name string, prefix string) string {
	// nolint: gosec
	h := md5.New()
	_, _ = h.Write([]byte(name))
	v := hex.EncodeToString(h.Sum(nil))
	return filepath.Join(s.location, fmt.Sprintf("%s-%s", prefix, v))
}

const requestIDLenSafe = 8

func (s *JSONDB) newFile(
	dagFile string, t time.Time, requestID string,
) (string, error) {
	if dagFile == "" {
		return "", errDAGFileEmpty
	}
	return fmt.Sprintf(
		"%s.%s.%s.dat",
		s.prefixWithDirectory(dagFile),
		t.Format("20060102.15:04:05.000"),
		util.TruncString(requestID, requestIDLenSafe),
	), nil
}

func (store *JSONDB) latestToday(
	dagFile string,
	day time.Time,
	latestStatusToday bool,
) (string, error) {
	var (
		ret     []string
		pattern string
	)
	if latestStatusToday {
		pattern = fmt.Sprintf(
			"%s.%s*.*.dat", store.prefixWithDirectory(dagFile), day.Format("20060102"),
		)
	} else {
		pattern = fmt.Sprintf("%s.*.*.dat", store.prefixWithDirectory(dagFile))
	}
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", persistence.ErrNoStatusDataToday
	}
	ret = filterLatest(matches, 1)

	if len(ret) == 0 {
		return "", persistence.ErrNoStatusData
	}
	return ret[0], err
}

func (*JSONDB) latest(pattern string, n int) []string {
	matches, err := filepath.Glob(pattern)
	var ret = []string{}
	if err == nil || len(matches) >= 0 {
		ret = filterLatest(matches, n)
	}
	return ret
}

const extDat = ".dat"

func (s *JSONDB) globPattern(dagFile string) string {
	return s.prefixWithDirectory(dagFile) + "*" + extDat
}

func (s *JSONDB) prefixWithDirectory(dagFile string) string {
	p := prefix(dagFile)
	return filepath.Join(s.getDirectory(dagFile, p), p)
}

func ParseFile(file string) (*model.Status, error) {
	f, err := os.Open(file)
	if err != nil {
		log.Printf("failed to open file. err: %v", err)
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	var (
		offset int64
		ret    *model.Status
	)
	for {
		line, err := readLineFrom(f, offset)
		if err == io.EOF {
			if ret == nil {
				return nil, err
			}
			return ret, nil
		} else if err != nil {
			return nil, err
		}
		offset += int64(len(line)) + 1 // +1 for newline
		if len(line) > 0 {
			var m *model.Status
			m, err = model.StatusFromJSON(string(line))
			if err == nil {
				ret = m
				continue
			}
		}
	}
}

func filterLatest(files []string, n int) []string {
	if len(files) == 0 {
		return []string{}
	}
	sort.Slice(files, func(i, j int) bool {
		t1 := timestamp(files[i])
		t2 := timestamp(files[j])
		return t1 > t2
	})
	ret := make([]string, 0, n)
	for i := 0; i < n && i < len(files); i++ {
		ret = append(ret, files[i])
	}
	return ret
}

var rTimestamp = regexp.MustCompile(`2\d{7}.\d{2}:\d{2}:\d{2}`)

func timestamp(file string) string {
	return rTimestamp.FindString(file)
}

func readLineFrom(f *os.File, offset int64) ([]byte, error) {
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	var ret []byte
	for {
		b, isPrefix, err := r.ReadLine()
		if err == io.EOF {
			return ret, err
		} else if err != nil {
			log.Printf("read line failed. %s", err)
			return nil, err
		}
		ret = append(ret, b...)
		if !isPrefix {
			break
		}
	}
	return ret, nil
}

func prefix(dagFile string) string {
	return strings.TrimSuffix(filepath.Base(dagFile), filepath.Ext(dagFile))
}
