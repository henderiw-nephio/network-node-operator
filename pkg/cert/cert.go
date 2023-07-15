package cert

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
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

/*
	func GetCertificateData(dir, profile string) (*CertData, error) {
		certData := &CertData{
			ProfileName: profile,
		}
		fmt.Printf("getCertificateData: %s\n", dir)
		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("getCertificateData error: %s\n", err.Error())
			return nil, err
		}
		for _, f := range files {
			fmt.Printf("filename: %s\n", f.Name())
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
*/
func GetCertificateData(secret *corev1.Secret, profile string) (*CertData, error) {
	certFiles := []string{"ca.crt", "tls.crt", "tls.key"}
	for _, certFile := range certFiles {
		fmt.Println("certFile:\n", secret.Data[certFile])
	}
	return nil, fmt.Errorf("not implemented yet")

	/*

		certData := &CertData{
			ProfileName: profile,
		}
		fmt.Printf("getCertificateData: %s\n", dir)

		certFiles := []string{"ca.crt", "tls.crt", "tls.key"}
		for _, certFile := range certFiles {
			var found bool
			b, err := os.ReadFile(filepath.Join(dir, certFile))
			if err != nil {
				fmt.Printf("getCertificateData: err: %s\n", err.Error())
				return nil, err
			}
			switch certFile {
			case "ca.crt":
				certData.CA, found = getStringInBetween(string(b), caStartMarker, caEndMarker, true)
				if !found {
					return nil, fmt.Errorf("cannot get the ca string")
				}
			case "tls.crt":
				certData.Cert, found = getStringInBetween(string(b), certStartMarker, certEndMarker, true)
				if !found {
					return nil, fmt.Errorf("cannot get the tls cert string")
				}
			case "tls.key":
				certData.Key, found = getStringInBetween(string(b), keyStartMarker, keyEndMarker, false)
				if !found {
					return nil, fmt.Errorf("cannot get the tls key string")
				}
				certData.Key = strings.ReplaceAll(certData.Key, "\n", "")
			}
		}
		return certData, nil
	*/
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
