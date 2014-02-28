package dry

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/zlib"
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/crc64"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	// "strconv"
	"strings"
	"time"
)

func FileBufferedReader(filenameOrURL string) (io.Reader, error) {
	data, err := FileGetBytes(filenameOrURL)
	if err != nil {
		return nil, err
	}
	return BytesReader(data), nil
}

func FileGetBytes(filenameOrURL string, headerTimeout ...time.Duration) ([]byte, error) {
	if strings.Contains(filenameOrURL, "://") {
		if strings.Index(filenameOrURL, "file://") == 0 {
			filenameOrURL = filenameOrURL[len("file://"):]
		} else {
			client := http.DefaultClient
			if len(headerTimeout) > 0 {
				client = &http.Client{
					Transport: &http.Transport{
						ResponseHeaderTimeout: headerTimeout[0],
					},
				}
			}
			r, err := client.Get(filenameOrURL)
			if err != nil {
				return nil, err
			}
			defer r.Body.Close()
			return ioutil.ReadAll(r.Body)
		}
	}
	return ioutil.ReadFile(filenameOrURL)
}

func FileSetBytes(filename string, data []byte) error {
	return ioutil.WriteFile(filename, data, 0)
}

func FileAppendBytes(filename string, data []byte) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func FileGetString(filenameOrURL string, headerTimeout ...time.Duration) (string, error) {
	bytes, err := FileGetBytes(filenameOrURL, headerTimeout...)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func FileSetString(filename string, data string) error {
	return FileSetBytes(filename, []byte(data))
}

func FileAppendString(filename string, data string) error {
	return FileAppendBytes(filename, []byte(data))
}

func FileGetJSON(filenameOrURL string, headerTimeout ...time.Duration) (result interface{}, err error) {
	err = FileUnmarshallJSON(filenameOrURL, &result, headerTimeout...)
	return result, err
}

func FileUnmarshallJSON(filenameOrURL string, result interface{}, headerTimeout ...time.Duration) error {
	data, err := FileGetBytes(filenameOrURL, headerTimeout...)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

func FileSetJSON(filename string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return FileSetBytes(filename, bytes)
}

func FileGetXML(filenameOrURL string, headerTimeout ...time.Duration) (result interface{}, err error) {
	err = FileUnmarshallXML(filenameOrURL, &result, headerTimeout...)
	return result, err
}

func FileUnmarshallXML(filenameOrURL string, result interface{}, headerTimeout ...time.Duration) error {
	data, err := FileGetBytes(filenameOrURL, headerTimeout...)
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, result)
}

func FileSetXML(filename string, data interface{}) error {
	bytes, err := xml.Marshal(data)
	if err != nil {
		return err
	}
	return FileSetBytes(filename, bytes)
}

func FileGetCSV(filenameOrURL string, headerTimeout ...time.Duration) ([][]string, error) {
	data, err := FileGetBytes(filenameOrURL, headerTimeout...)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(bytes.NewBuffer(data))
	return reader.ReadAll()
}

func FileSetCSV(filename string, records [][]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	return writer.WriteAll(records)
}

func FileGetConfig(filenameOrURL string, headerTimeout ...time.Duration) (map[string]string, error) {
	data, err := FileGetBytes(filenameOrURL, headerTimeout...)
	if err != nil {
		return nil, err
	}

	lines := bytes.Split(data, []byte("\n"))
	config := make(map[string]string, len(lines))
	for _, line := range lines {
		kv := bytes.SplitN(line, []byte("="), 2)
		if len(kv) < 2 {
			continue
		}
		key := string(bytes.TrimSpace(kv[0]))
		if len(key) == 0 || key[0] == '#' {
			continue
		}
		value := string(bytes.TrimSpace(kv[1]))
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		config[key] = value
	}

	return config, nil
}

func FileSetConfig(filename string, config map[string]string) error {
	var buffer bytes.Buffer
	for key, value := range config {
		if strings.ContainsRune(key, '=') {
			return fmt.Errorf("Key '%s' contains '='", key)
		}
		fmt.Fprintf(&buffer, "%s=%s\n", key, value)
	}
	return FileSetBytes(filename, buffer.Bytes())
}

// FileGetLastLine reads the last line from a file.
// In case of a network file, the whole file is read.
// In case of a local file, the last 64kb are read,
// so if the last line is longer than 64kb it is not returned completely.
// The first optional headerTimeout is used for network files only.
func FileGetLastLine(filenameOrURL string, headerTimeout ...time.Duration) (line string, err error) {
	if strings.Index(filenameOrURL, "file://") == 0 {
		return FileGetLastLine(filenameOrURL[len("file://"):])
	}

	var data []byte

	if strings.Contains(filenameOrURL, "://") {
		data, err = FileGetBytes(filenameOrURL, headerTimeout...)
		if err != nil {
			return "", err
		}
	} else {
		file, err := os.Open(filenameOrURL)
		if err != nil {
			return "", err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return "", err
		}
		if start := info.Size() - 64*1024; start > 0 {
			file.Seek(start, os.SEEK_SET)
		}
		data, err = ioutil.ReadAll(file)
		if err != nil {
			return "", err
		}
	}

	pos := bytes.LastIndex(data, []byte{'\n'})
	return string(data[pos+1:]), nil
}

// func FileTail(filenameOrURL string, numLines int, headerTimeout ...time.Duration) (lines []string, err error) {
// 	if strings.Index(filenameOrURL, "file://") == 0 {
// 		filenameOrURL = filenameOrURL[len("file://"):]
// 	} else if strings.Contains(filenameOrURL, "://") {
// 		data, err := FileGetBytes(filenameOrURL, headerTimeout...)
// 		if err != nil {
// 			return nil, err
// 		}
// 		lines, _ := BytesTail(data, numLines)
// 		return lines, nil
// 	}

// 	// data := make([]byte, 0, 1024*256)

// 	// file, err := os.Open(filenameOrURL)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// defer file.Close()
// 	// info, err := file.Stat()
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// if start := info.Size() - 64*1024; start > 0 {
// 	// 	file.Seek(start, os.SEEK_SET)
// 	// }
// 	// data, err = ioutil.ReadAll(file)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	return lines, nil

// }

// FileTimeModified returns the modified time of a file,
// or the zero time value in case of an error.
func FileTimeModified(filename string) time.Time {
	info, err := os.Stat(filename)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func FileIsDir(dirname string) bool {
	info, err := os.Stat(dirname)
	return err == nil && info.IsDir()
}

func FileFind(searchDirs []string, filename string) (filePath string, found bool) {
	for _, dir := range searchDirs {
		filePath = path.Join(dir, filename)
		if FileExists(filePath) {
			return filePath, true
		}
	}
	return "", false
}

func FileTouch(filename string) error {
	if FileExists(filename) {
		now := time.Now()
		return os.Chtimes(filename, now, now)
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	return file.Close()
}

func FileMD5String(filenameOrURL string) (string, error) {
	sum, err := FileMD5Bytes(filenameOrURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sum), nil
}

func FileMD5Bytes(filenameOrURL string) ([]byte, error) {
	data, err := FileGetBytes(filenameOrURL)
	if err != nil {
		return nil, err
	}
	hash := md5.New()
	_, err = io.Copy(hash, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

var crc64Table *crc64.Table

func FileCRC64(filenameOrURL string) (uint64, error) {
	data, err := FileGetBytes(filenameOrURL)
	if err != nil {
		return 0, err
	}
	if crc64Table == nil {
		crc64Table = crc64.MakeTable(crc64.ECMA)
	}
	return crc64.Checksum(data, crc64Table), nil
}

func FileGetInflate(filenameOrURL string) ([]byte, error) {
	data, err := FileGetBytes(filenameOrURL)
	if err != nil {
		return nil, err
	}
	reader := flate.NewReader(bytes.NewBuffer(data))
	defer reader.Close()
	return ioutil.ReadAll(reader)
}

func FileSetDeflate(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	fileBuf := bufio.NewWriter(file)
	defer fileBuf.Flush()
	writer, err := flate.NewWriter(fileBuf, flate.BestCompression)
	if err != nil {
		return err
	}
	err = WriteAllBytes(data, writer)
	if err != nil {
		return err
	}
	return writer.Close()
}

func FileGetGz(filenameOrURL string) ([]byte, error) {
	data, err := FileGetBytes(filenameOrURL)
	if err != nil {
		return nil, err
	}
	reader, err := zlib.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return ioutil.ReadAll(reader)
}

func FileSetGz(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	fileBuf := bufio.NewWriter(file)
	defer fileBuf.Flush()
	writer, err := zlib.NewWriterLevel(fileBuf, zlib.BestCompression)
	if err != nil {
		return err
	}
	err = WriteAllBytes(data, writer)
	if err != nil {
		return err
	}
	return writer.Close()
}

// FileSize returns the size of a file or zero in case of an error.
func FileSize(filename string) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return info.Size()
}

func FilePrintf(filename, format string, args ...interface{}) error {
	file, err := os.OpenFile(filename, os.O_WRONLY, 0660)
	if err == nil {
		_, err = fmt.Fprintf(file, format, args...)
		file.Close()
	}
	return err
}

func FileAppendPrintf(filename, format string, args ...interface{}) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err == nil {
		_, err = fmt.Fprintf(file, format, args...)
		file.Close()
	}
	return err
}

func FileScanf(filename, format string, args ...interface{}) error {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0660)
	if err == nil {
		_, err = fmt.Fscanf(file, format, args...)
		file.Close()
	}
	return err
}

func ListDir(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdirnames(-1)
}

func ListDirFiles(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileInfos, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(fileInfos))
	for i := range fileInfos {
		if !fileInfos[i].IsDir() {
			result = append(result, fileInfos[i].Name())
		}
	}
	return result, nil
}

func ListDirDirectories(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileInfos, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(fileInfos))
	for i := range fileInfos {
		if fileInfos[i].IsDir() {
			result = append(result, fileInfos[i].Name())
		}
	}
	return result, nil
}
