package objects

import (
	"encoding/xml"
	"os"

	"github.com/aidenappl/openbucket-go/types"
)

func RetrieveObjectMetadata(filePath string) *types.ObjectMetadata {
	f, err := os.Open(filePath + ".obmeta")
	if err != nil {
		return nil
	}
	defer f.Close()

	var md types.ObjectMetadata
	if err := xml.NewDecoder(f).Decode(&md); err != nil {
		return nil
	}

	return &md
}
