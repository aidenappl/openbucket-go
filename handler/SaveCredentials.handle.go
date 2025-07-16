package handler

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"

	"log"
	"os"
	"time"

	"github.com/aidenappl/openbucket-go/types"
)

func SaveCredentials(creds *types.Authorization) error {
	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}

	filePath := "authorizations.xml"

	var existingAuthorizations types.Authorizations
	if _, err := os.Stat(filePath); err == nil {

		xmlData, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read existing XML file: %v", err)
		}

		err = xml.Unmarshal(xmlData, &existingAuthorizations)
		if err != nil {
			return fmt.Errorf("failed to unmarshal existing XML data: %v", err)
		}
	}

	if creds.KeyID == "" || creds.SecretKey == "" {
		return fmt.Errorf("credentials KeyID and SecretKey cannot be empty")
	}

	if creds.Name == "" {
		return fmt.Errorf("credentials Name cannot be empty")
	}

	if creds.DateCreated.IsZero() {
		creds.DateCreated = time.Now()
	}

	existingAuthorizations.Authorizations = append(existingAuthorizations.Authorizations, *creds)

	xmlData, err := xml.MarshalIndent(existingAuthorizations, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated XML data: %v", err)
	}

	err = ioutil.WriteFile(filePath, xmlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated XML file: %v", err)
	}

	log.Printf("Credentials saved to %s\n", filePath)
	return nil
}
