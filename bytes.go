package dry

import (
	"bytes"
	"compress/flate"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

func BytesReader(data interface{}) io.Reader {
	switch s := data.(type) {
	case io.Reader:
		return s
	case []byte:
		return bytes.NewReader(s)
	case string:
		return strings.NewReader(s)
	case fmt.Stringer:
		strings.NewReader(s.String())
	case error:
		strings.NewReader(s.Error())
	}
	return nil
}

func BytesMD5(data string) string {
	hash := md5.New()
	hash.Write([]byte(data))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func BytesEncodeBase64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func BytesDecodeBase64(base64Str string) string {
	result, _ := base64.StdEncoding.DecodeString(base64Str)
	return string(result)
}

func BytesEncodeHex(str string) string {
	return hex.EncodeToString([]byte(str))
}

func BytesDecodeHex(hexStr string) string {
	result, _ := hex.DecodeString(hexStr)
	return string(result)
}

func BytesZip(str string) []byte {
	var buf bytes.Buffer
	writer, _ := flate.NewWriter(&buf, flate.BestCompression)
	writer.Write([]byte(str))
	writer.Close()
	return buf.Bytes()
}

func BytesUnzip(zipped []byte) string {
	reader := flate.NewReader(bytes.NewBuffer(zipped))
	result, _ := ioutil.ReadAll(reader)
	return string(result)
}

// BytesHead returns at most numLines from data starting at the beginning.
// A slice of the remaining data is returned as rest.
// \n is used to detect line ends, a preceding \r will be stripped away.
// BytesHead resembles the Unix head command.
func BytesHead(data []byte, numLines int) (lines []string, rest []byte) {
	if numLines <= 0 {
		panic("numLines must be greater than zero")
	}
	lines = make([]string, 0, numLines)
	begin := 0
	for i := range data {
		if data[i] == '\n' {
			end := i
			if i > 0 && data[i-1] == '\r' {
				end--
			}
			lines = append(lines, string(data[begin:end]))
			begin = i + 1
			if len(lines) == numLines {
				break
			}
		}
	}
	return lines, data[begin:]
}

// BytesTail returns at most numLines from the end of data.
// A slice of the remaining data before lines is returned as rest.
// \n is used to detect line ends, a preceding \r will be stripped away.
// BytesTail resembles the Unix tail command.
func BytesTail(data []byte, numLines int) (lines []string, rest []byte) {
	if numLines <= 0 {
		panic("numLines must be greater than zero")
	}
	lines = make([]string, 0, numLines)
	begin := 0
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == '\n' {
			end := i
			if i > 0 && data[i-1] == '\r' {
				end--
			}
			lines = append(lines, string(data[begin:end]))
			begin = i + 1
			if len(lines) == numLines {
				break
			}
		}
	}
	return lines, data[:begin]
}
