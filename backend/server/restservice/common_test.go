package restservice

import (
	"reflect"
)

// Extracts HTTP status code from the response received via the
// REST API.
func getStatusCode(rsp interface{}) int {
	code := int(reflect.ValueOf(rsp).FieldByName("_statusCode").Int())
	return code
}
