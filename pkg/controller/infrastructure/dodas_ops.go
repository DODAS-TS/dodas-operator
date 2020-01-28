package infrastructure

import (
	"fmt"
	"strings"

	dodasv1alpha1 "github.com/dodas-ts/dodas-operator/pkg/apis/dodas/v1alpha1"
)

// DeleteInf wraps the actions for deleting infrastructure
func DeleteInf(instance *dodasv1alpha1.Infrastructure) error {
	authHeader := PrepareAuthHeaders(instance)

	request := Request{
		URL:         string(instance.Spec.ImAuth.Host) + "/" + instance.Status.InfID,
		RequestType: "DELETE",
		Headers: map[string]string{
			"Authorization": authHeader,
		},
	}

	body, statusCode, err := MakeRequest(request)
	if err != nil {
		return err
	}

	if statusCode != 200 {
		return fmt.Errorf("Status code: %d\n Body: %s", statusCode, body)
	}

	return nil

}

// CreateInf wraps the actions for creating an infrastructure
func CreateInf(instance *dodasv1alpha1.Infrastructure, template []byte) (string, error) {
	authHeader := PrepareAuthHeaders(instance)

	request := Request{
		URL:         string(instance.Spec.ImAuth.Host),
		RequestType: "POST",
		Headers: map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "text/yaml",
		},
		Content: []byte(template),
	}

	body, statusCode, err := MakeRequest(request)
	if err != nil {
		return "", err
	}

	// save infID in status or the error
	if statusCode != 200 {
		return "", fmt.Errorf("Status code: %d\n Body: %s", statusCode, body)
	}

	stringSplit := strings.Split(string(body), "/")
	return stringSplit[len(stringSplit)-1], nil

}
