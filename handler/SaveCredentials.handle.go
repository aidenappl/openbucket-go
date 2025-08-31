package handler

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"log"
	"os"
	"time"

	"github.com/aidenappl/openbucket-go/types"
)

func SaveCredentials(creds *types.Authorization) error {
	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}

	filePath := "buckets/authorizations.xml"

	existingAuthorizations, err := loadExistingAuthorizations(filePath)
	if err != nil {
		return err
	}

	if existingAuthorizations == nil {
		err = createBlankAuthorizationsFile(filePath)
		if err != nil {
			return err
		}
		existingAuthorizations, err = loadExistingAuthorizations(filePath)
		if err != nil {
			return err
		}
		if existingAuthorizations == nil {
			return fmt.Errorf("failed to initialize authorizations")
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

	xmlData = append([]byte(xml.Header), xmlData...)
	err = ioutil.WriteFile(filePath, xmlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated XML file: %v", err)
	}

	log.Printf("Credentials saved to %s\n", filePath)
	return nil
}

// loadExistingAuthorizations reads and unmarshals existing authorizations from file
// Returns empty Authorizations struct if file doesn't exist
func loadExistingAuthorizations(filePath string) (*types.Authorizations, error) {
	var authorizations types.Authorizations

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, return empty authorizations
		return nil, nil
	}

	// File exists, read and unmarshal it
	xmlData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing XML file: %v", err)
	}

	err = xml.Unmarshal(xmlData, &authorizations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal existing XML data: %v", err)
	}

	return &authorizations, nil
}

// createBlankAuthorizationsFile creates a new empty authorizations file
func createBlankAuthorizationsFile(filePath string) error {
	blankAuthorizations := types.Authorizations{}

	// atomic write
	tmp := filePath + ".tmp"
	if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
		return err
	}
	buf, err := xml.MarshalIndent(blankAuthorizations, "", "  ")
	if err != nil {
		return err
	}
	buf = append([]byte(xml.Header), buf...)
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filePath)
}
