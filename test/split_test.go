package test

import (
	"net/url"
	"strings"
	"testing"
)

func TestSplit(t *testing.T) {

	state := "state=knexus%24success%3Dhttps%3A%2F%2Fknexus.xyz%24fail%3Dhttps%3A%2F%2Fknexus.xyz"
	decodedURL, err := url.QueryUnescape(state)

	if err != nil {
		t.Errorf("11")
		return
	}

	t.Logf("{%v}", decodedURL)

	result := strings.Split(decodedURL, "$")

	success := result[1]
	fail := result[2]
	t.Logf("{%s}", strings.Replace(fail, "fail=", "", 1))
	t.Logf("{%s}", strings.Replace(success, "success=", "", 1))

	// str := fmt.Sprintf("%v", result)

	// arr := []int{1, 2, 3, 4, 5}
	t.Logf("{%v}", result[0])
}
