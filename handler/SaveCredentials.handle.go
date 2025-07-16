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

	// Check if the file exists and open it for reading
	var existingAuthorizations types.Authorizations
	if _, err := os.Stat(filePath); err == nil {
		// File exists, read the existing data
		xmlData, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read existing XML file: %v", err)
		}

		// Unmarshal the existing XML data into the Authorizations struct
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

	if creds.DateCreated == "" {
		creds.DateCreated = time.Now().Format(time.RFC3339)
	}

	// Append the new credentials to the existing list
	existingAuthorizations.Authorizations = append(existingAuthorizations.Authorizations, *creds)

	// Marshal the updated data back to XML
	xmlData, err := xml.MarshalIndent(existingAuthorizations, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated XML data: %v", err)
	}

	// Write the updated XML back to the file
	err = ioutil.WriteFile(filePath, xmlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated XML file: %v", err)
	}

	log.Printf("Credentials saved to %s\n", filePath)
	return nil
}
