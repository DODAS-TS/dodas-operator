package infrastructure

import (
	"fmt"
	"reflect"
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

var decodeFields = map[string]string{
	"ID":            "id",
	"Type":          "type",
	"Username":      "username",
	"Password":      "password",
	"Token":         "token",
	"Host":          "host",
	"Tenant":        "tenant",
	"AuthURL":       "auth_url",
	"AuthVersion":   "auth_version",
	"Domain":        "domain",
	"ServiceRegion": "service_region",
}

// PrepareAuthHeaders ..
func PrepareAuthHeaders(clientConf *dodasv1alpha1.Infrastructure) string {

	var authHeaderCloudList []string

	fields := reflect.TypeOf(clientConf.Spec.CloudAuth)
	values := reflect.ValueOf(clientConf.Spec.CloudAuth)

	// TODO: use go templates!

	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		value := values.Field(i)

		if value.Interface() != "" {
			keyTemp := fmt.Sprintf("%v = %v", decodeFields[field.Name], value)
			authHeaderCloudList = append(authHeaderCloudList, keyTemp)
		}
	}

	authHeaderCloud := strings.Join(authHeaderCloudList, ";")

	var authHeaderIMList []string

	fields = reflect.TypeOf(clientConf.Spec.ImAuth)
	values = reflect.ValueOf(clientConf.Spec.ImAuth)

	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		if decodeFields[field.Name] != "host" {
			value := values.Field(i)
			if value.Interface() != "" {
				keyTemp := fmt.Sprintf("%v = %v", decodeFields[field.Name], value.Interface())
				authHeaderIMList = append(authHeaderIMList, keyTemp)
			}
		}
	}

	authHeaderIM := strings.Join(authHeaderIMList, ";")

	authHeader := authHeaderCloud + "\\n" + authHeaderIM

	//fmt.Printf(authHeader)

	return authHeader
}
