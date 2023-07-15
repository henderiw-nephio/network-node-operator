package cert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	keyStartMarker  = "-----BEGIN RSA PRIVATE KEY-----"
	keyEndMarker    = "-----END RSA PRIVATE KEY-----"
	certStartMarker = "-----BEGIN CERTIFICATE-----"
	certEndMarker   = "-----END CERTIFICATE-----"
	caStartMarker   = "-----BEGIN CERTIFICATE-----"
	caEndMarker     = "-----END CERTIFICATE-----"
)

type CertData struct {
	ProfileName string
	CA          string
	Cert        string
	Key         string
}

func GetCertificateData(dir, profile string) (*CertData, error) {
	certData := &CertData{
		ProfileName: profile,
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if !f.IsDir() {
			b, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				return nil, err
			}
			var found bool
			if f.Name() == "ca.crt" {
				certData.CA, found = getStringInBetween(string(b), caStartMarker, caEndMarker, true)
				if !found {
					return nil, fmt.Errorf("cannot get the ca string")
				}
			}
			if f.Name() == "tls.crt" {
				certData.Cert, found = getStringInBetween(string(b), certStartMarker, certEndMarker, true)
				if !found {
					return nil, fmt.Errorf("cannot get the cert string")
				}
			}
			if f.Name() == "tls.key" {
				certData.Key, found = getStringInBetween(string(b), keyStartMarker, keyEndMarker, false)
				if !found {
					return nil, fmt.Errorf("cannot get the key string")
				}
				certData.Key = strings.ReplaceAll(certData.Key, "\n", "")
			}
		}
	}
	return certData, nil
}

// GetStringInBetween returns a string between the start/end markers with markers either included or excluded
func getStringInBetween(str string, start, end string, include bool) (result string, found bool) {
	// start index
	sidx := strings.Index(str, start)
	if sidx == -1 {
		return "", false
	}

	// forward start index if we don't want to include the markers
	if !include {
		sidx += len(start)
	}

	newS := str[sidx:]

	// end index
	eidx := strings.Index(newS, end)
	if eidx == -1 {
		return "", false
	}
	// to include the end marker, increment the end index up till its length
	if include {
		eidx += len(end)
	}

	return newS[:eidx], true
}
