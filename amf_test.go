
package rtmp

import (
	"testing"
	"encoding/base64"
	"bytes"
	"fmt"
)

var (
	data = `AgAHY29ubmVjdAA/8AAAAAAAAAMAA2FwcAIABW15YXBwAAhmbGFzaFZlcgIAEE1BQyAxMSw1LDUwMiwxNDkABnN3ZlVybAIAJmh0dHA6Ly9sb2NhbGhvc3Q6ODA4MS9zd2YvandwbGF5ZXIuc3dmAAV0Y1VybAIAFnJ0bXA6Ly9sb2NhbGhvc3QvbXlhcHAABGZwYWQBAAAMY2FwYWJpbGl0aWVzAEBt4AAAAAAAAAthdWRpb0NvZGVjcwBAq+4AAAAAAAALdmlkZW9Db2RlY3MAQG+AAAAAAAAADXZpZGVvRnVuY3Rpb24AP/AAAAAAAAAAB3BhZ2VVcmwCABpodHRwOi8vbG9jYWxob3N0OjgwODEvc3dmLwAOb2JqZWN0RW5jb2RpbmcAAAAAAAAAAAAAAAk=`
)

func TestHal(t *testing.T) {
	dec := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(data))
	r := NewAMFReader(dec)
	obj := r.ReadAMF()
	fmt.Printf("%v\n", obj)
}

